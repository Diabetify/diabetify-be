package services

import (
	"context"
	"diabetify/internal/ml"
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type PredictionJobWorker struct {
	// Repositories
	jobRepo      repository.PredictionJobRepository
	predRepo     repository.PredictionRepository
	userRepo     repository.UserRepository
	profileRepo  repository.UserProfileRepository
	activityRepo repository.ActivityRepository

	// ML Client
	mlClient ml.MLClient

	// Job processing
	jobQueue    chan models.PredictionJobRequest
	workerCount int
	stopChan    chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex

	// RabbitMQ for async responses
	conn            *amqp.Connection
	responseChannel *amqp.Channel
	pendingJobs     map[string]chan *models.PredictionResponse
	pendingJobsMu   sync.RWMutex

	// Configuration
	maxJobTimeout   time.Duration
	maxConcurrency  int
	cleanupInterval time.Duration
}

func NewPredictionJobWorker(
	jobRepo repository.PredictionJobRepository,
	predRepo repository.PredictionRepository,
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	activityRepo repository.ActivityRepository,
	mlClient ml.MLClient,
	workerCount int,
) *PredictionJobWorker {
	if workerCount <= 0 {
		workerCount = 3 // Default worker count
	}

	return &PredictionJobWorker{
		jobRepo:         jobRepo,
		predRepo:        predRepo,
		userRepo:        userRepo,
		profileRepo:     profileRepo,
		activityRepo:    activityRepo,
		mlClient:        mlClient,
		jobQueue:        make(chan models.PredictionJobRequest, 100),
		workerCount:     workerCount,
		stopChan:        make(chan struct{}),
		pendingJobs:     make(map[string]chan *models.PredictionResponse),
		maxJobTimeout:   5 * time.Minute,
		maxConcurrency:  10,
		cleanupInterval: 30 * time.Minute,
	}
}

// RabbitMQPredictionResponse is used specifically for parsing RabbitMQ messages
type RabbitMQPredictionResponse struct {
	Prediction    float64                           `json:"prediction"`
	Explanation   map[string]map[string]interface{} `json:"explanation"`
	ElapsedTime   float64                           `json:"elapsed_time"`
	Timestamp     string                            `json:"timestamp"`
	CorrelationID string                            `json:"correlation_id"`
	Error         *string                           `json:"error"`
}

// parseTimestamp safely parses the Python timestamp format
func parseTimestamp(timestampStr string) time.Time {
	// List of formats to try (including the one from Python)
	formats := []string{
		"2006-01-02T15:04:05.000000",  // Python's format: 2025-07-11T00:56:07.576363
		"2006-01-02T15:04:05",         // Without microseconds
		time.RFC3339,                  // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,              // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02T15:04:05Z",        // UTC format
		"2006-01-02T15:04:05.000Z",    // UTC with milliseconds
		"2006-01-02T15:04:05.000000Z", // UTC with microseconds
		"2006-01-02 15:04:05",         // Space separated
	}

	for _, format := range formats {
		if parsedTime, err := time.Parse(format, timestampStr); err == nil {
			return parsedTime
		}
	}

	return time.Now()
}

// convertToModelsResponse converts RabbitMQPredictionResponse to models.PredictionResponse
func convertToModelsResponse(rabbitResponse *RabbitMQPredictionResponse) *models.PredictionResponse {
	// Convert explanation map
	explanation := make(map[string]models.ExplanationItem)

	for featureName, featureData := range rabbitResponse.Explanation {
		explanationItem := models.ExplanationItem{}

		if shap, ok := featureData["shap"].(float64); ok {
			explanationItem.Shap = shap
		}
		if contribution, ok := featureData["contribution"].(float64); ok {
			explanationItem.Contribution = contribution
		}
		if impact, ok := featureData["impact"].(float64); ok {
			explanationItem.Impact = int(impact)
		}

		explanation[featureName] = explanationItem
	}

	return &models.PredictionResponse{
		Prediction:  rabbitResponse.Prediction,
		Explanation: explanation,
		ElapsedTime: rabbitResponse.ElapsedTime,
		Timestamp:   parseTimestamp(rabbitResponse.Timestamp),
	}
}

// ========== WORKER LIFECYCLE ==========

func (w *PredictionJobWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	// Setup RabbitMQ response consumer, ignoring error as in original logic
	_ = w.setupRabbitMQConsumer()

	// Start worker goroutines
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}

	// Start job recovery routine (process any pending jobs from database)
	w.wg.Add(1)
	go w.recoverPendingJobs()

	// Start cleanup routine
	w.wg.Add(1)
	go w.cleanupRoutine()
}

func (w *PredictionJobWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	// Close RabbitMQ connection
	if w.responseChannel != nil {
		w.responseChannel.Close()
	}
	if w.conn != nil {
		w.conn.Close()
	}

	close(w.stopChan)
	w.wg.Wait()
}

// ========== RABBITMQ SETUP ==========
func (w *PredictionJobWorker) setupRabbitMQConsumer() error {
	// Connect to RabbitMQ
	var err error
	w.conn, err = amqp.Dial("amqp://admin:password123@localhost:5672/")
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	w.responseChannel, err = w.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}

	// Declare the response queue
	_, err = w.responseChannel.QueueDeclare(
		"ml.prediction.hybrid_response", // name
		true,                            // durable
		false,                           // delete when unused
		false,                           // exclusive
		false,                           // no-wait
		nil,                             // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Start consuming responses - THIS IS THE ONLY CONSUMER NOW
	msgs, err := w.responseChannel.Consume(
		"ml.prediction.hybrid_response", // queue
		"unified_consumer",              // consumer - unique tag
		false,                           // auto-ack
		false,                           // exclusive
		false,                           // no-local
		false,                           // no-wait
		nil,                             // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %v", err)
	}

	// Start unified response handler
	w.wg.Add(1)
	go w.handleResponses(msgs)

	return nil
}

func (w *PredictionJobWorker) handleResponses(msgs <-chan amqp.Delivery) {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopChan:
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			correlationID := msg.CorrelationId

			var rabbitResponse RabbitMQPredictionResponse
			if err := json.Unmarshal(msg.Body, &rabbitResponse); err != nil {
				msg.Nack(false, false)
				continue
			}

			modelResponse := convertToModelsResponse(&rabbitResponse)

			delivered := false

			if hybridClient, ok := w.mlClient.(interface {
				DeliverResponse(correlationID string, response *models.PredictionResponse) bool
			}); ok {
				if hybridClient.DeliverResponse(correlationID, modelResponse) {
					delivered = true
				}
			}

			if !delivered {
				w.pendingJobsMu.Lock()
				localResponseChan, exists := w.pendingJobs[correlationID]
				if exists {
					delete(w.pendingJobs, correlationID)
					w.pendingJobsMu.Unlock()

					select {
					case localResponseChan <- modelResponse:
						delivered = true
					case <-time.After(2 * time.Second):
						// Timeout, delivered remains false
					}
				} else {
					w.pendingJobsMu.Unlock()
				}
			}

			_ = msg.Ack(false)
		}
	}
}

func (w *PredictionJobWorker) SubmitJob(jobRequest models.PredictionJobRequest) error {
	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return fmt.Errorf("job worker is not running")
	}
	w.mu.RUnlock()

	activeJobs, err := w.jobRepo.GetActiveJobsCount(jobRequest.UserID)
	if err != nil {
		return fmt.Errorf("failed to check active jobs: %w", err)
	}

	if activeJobs >= int64(w.maxConcurrency) {
		return fmt.Errorf("user has too many active jobs (%d/%d)", activeJobs, w.maxConcurrency)
	}

	select {
	case w.jobQueue <- jobRequest:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("job queue is full, try again later")
	}
}

// ========== WORKER IMPLEMENTATION ==========

func (w *PredictionJobWorker) worker(workerID int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopChan:
			return
		case jobRequest := <-w.jobQueue:
			w.processJob(jobRequest)
		}
	}
}

func (w *PredictionJobWorker) processJob(jobRequest models.PredictionJobRequest) {
	jobID := jobRequest.JobID
	userID := jobRequest.UserID

	ctx, cancel := context.WithTimeout(context.Background(), w.maxJobTimeout)
	defer cancel()

	if err := w.jobRepo.UpdateJobStatus(jobID, models.JobStatusProcessing, nil); err != nil {
		return
	}

	_ = w.jobRepo.UpdateJobProgress(jobID, 20, models.JobStepValidatingProfile)

	user, err := w.userRepo.GetUserByID(userID)
	if err != nil {
		errMsg := fmt.Sprintf("User not found: %v", err)
		w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	profile, err := w.profileRepo.FindByUserID(userID)
	if err != nil {
		errMsg := fmt.Sprintf("Profile not found: %v", err)
		w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	_ = w.jobRepo.UpdateJobProgress(jobID, 40, models.JobStepCalculatingFeatures)

	features, featureInfo, err := w.calculateFeaturesFromProfile(user, profile, userID, jobRequest.WhatIfInput)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to calculate features: %v", err)
		w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	_ = w.jobRepo.UpdateJobProgress(jobID, 50, "Submitting to ML service")

	correlationID := jobID

	responseChan := make(chan *models.PredictionResponse, 1)
	w.mlClient.RegisterPendingCall(correlationID, responseChan)

	defer func() {
		w.mlClient.UnregisterPendingCall(correlationID)
		close(responseChan)
	}()

	submitCtx, submitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer submitCancel()

	if err := w.mlClient.PredictAsync(submitCtx, correlationID, features); err != nil {
		errMsg := fmt.Sprintf("Failed to submit to ML service: %v", err)
		w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	_ = w.jobRepo.UpdateJobProgress(jobID, 60, "Waiting for ML service response")

	var response *models.PredictionResponse
	progressTicker := time.NewTicker(3 * time.Second)
	defer progressTicker.Stop()
	progress := 60

	for {
		select {
		case <-ctx.Done():
			errMsg := fmt.Sprintf("Timeout waiting for ML response: %v", ctx.Err())
			w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
			return

		case response = <-responseChan:
			if response != nil {
				goto processResponse
			}

		case <-progressTicker.C:
			if progress < 80 {
				progress += 2
				_ = w.jobRepo.UpdateJobProgress(jobID, progress, "Waiting for ML service response")
			}
		}
	}

processResponse:
	_ = w.jobRepo.UpdateJobProgress(jobID, 90, models.JobStepSavingResults)

	prediction := w.createPredictionRecord(userID, response, featureInfo)

	if err := w.predRepo.SavePrediction(prediction); err != nil {
		errMsg := fmt.Sprintf("Failed to save prediction: %v", err)
		w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	if jobRequest.WhatIfInput == nil {
		now := time.Now()
		_ = w.userRepo.UpdateLastPredictionTime(userID, &now)
	}

	if err := w.jobRepo.UpdateJobStatusWithResult(jobID, models.JobStatusCompleted, prediction.ID); err != nil {
		return
	}
}

// ========== HELPER METHODS ==========

func (w *PredictionJobWorker) createPredictionRecord(userID uint, response *models.PredictionResponse, featureInfo map[string]interface{}) *models.Prediction {
	getExplanation := func(key string) (float64, float64, float64) {
		if exp, exists := response.Explanation[key]; exists {
			return exp.Shap, exp.Contribution, float64(exp.Impact)
		}
		return 0.0, 0.0, 0.0
	}

	ageShap, ageContribution, ageImpact := getExplanation("age")
	bmiShap, bmiContribution, bmiImpact := getExplanation("BMI")
	brinkmanShap, brinkmanContribution, brinkmanImpact := getExplanation("brinkman_index")
	hypertensionShap, hypertensionContribution, hypertensionImpact := getExplanation("is_hypertension")
	cholesterolShap, cholesterolContribution, cholesterolImpact := getExplanation("is_cholesterol")
	bloodlineShap, bloodlineContribution, bloodlineImpact := getExplanation("is_bloodline")
	macrosomicShap, macrosomicContribution, macrosomicImpact := getExplanation("is_macrosomic_baby")
	smokingShap, smokingContribution, smokingImpact := getExplanation("smoking_status")
	activityShap, activityContribution, activityImpact := getExplanation("moderate_physical_activity_frequency")

	return &models.Prediction{
		UserID:    userID,
		RiskScore: response.Prediction,

		Age:             featureInfo["age"].(int),
		AgeShap:         ageShap,
		AgeContribution: ageContribution,
		AgeImpact:       ageImpact,

		BMI:             featureInfo["bmi"].(float64),
		BMIShap:         bmiShap,
		BMIContribution: bmiContribution,
		BMIImpact:       bmiImpact,

		BrinkmanScore:             featureInfo["brinkman_score"].(int),
		BrinkmanScoreShap:         brinkmanShap,
		BrinkmanScoreContribution: brinkmanContribution,
		BrinkmanScoreImpact:       brinkmanImpact,

		IsHypertension:             featureInfo["is_hypertension"].(bool),
		IsHypertensionShap:         hypertensionShap,
		IsHypertensionContribution: hypertensionContribution,
		IsHypertensionImpact:       hypertensionImpact,

		IsCholesterol:             featureInfo["is_cholesterol"].(bool),
		IsCholesterolShap:         cholesterolShap,
		IsCholesterolContribution: cholesterolContribution,
		IsCholesterolImpact:       cholesterolImpact,

		IsBloodline:             featureInfo["is_bloodline"].(bool),
		IsBloodlineShap:         bloodlineShap,
		IsBloodlineContribution: bloodlineContribution,
		IsBloodlineImpact:       bloodlineImpact,

		IsMacrosomicBaby:             featureInfo["is_macrosomic_baby"].(int),
		IsMacrosomicBabyShap:         macrosomicShap,
		IsMacrosomicBabyContribution: macrosomicContribution,
		IsMacrosomicBabyImpact:       macrosomicImpact,

		SmokingStatus:             featureInfo["smoking_status"].(int),
		SmokingStatusShap:         smokingShap,
		SmokingStatusContribution: smokingContribution,
		SmokingStatusImpact:       smokingImpact,

		PhysicalActivityFrequency:             featureInfo["physical_activity_frequency"].(int),
		PhysicalActivityFrequencyShap:         activityShap,
		PhysicalActivityFrequencyContribution: activityContribution,
		PhysicalActivityFrequencyImpact:       activityImpact,
	}
}

// ========== BACKGROUND ROUTINES ==========

func (w *PredictionJobWorker) recoverPendingJobs() {
	defer w.wg.Done()

	time.Sleep(5 * time.Second)

	pendingJobs, err := w.jobRepo.GetPendingJobs(50)
	if err != nil {
		return
	}

	if len(pendingJobs) > 0 {
		for _, job := range pendingJobs {
			jobRequest := models.PredictionJobRequest{
				JobID:  job.ID,
				UserID: job.UserID,
			}

			select {
			case w.jobQueue <- jobRequest:
			case <-w.stopChan:
				return
			default:
			}
		}
	}
}

func (w *PredictionJobWorker) cleanupRoutine() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoffTime := time.Now().AddDate(0, 0, -7)
			_ = w.jobRepo.CleanupOldJobs(cutoffTime)
		case <-w.stopChan:
			return
		}
	}
}

// ========== FEATURE CALCULATION ==========

func (w *PredictionJobWorker) calculateFeaturesFromProfile(user *models.User, profile *models.UserProfile, userID uint, input *models.WhatIfInput) ([]float64, map[string]interface{}, error) {
	if user.DOB == nil {
		return nil, nil, fmt.Errorf("date of birth is required but not found")
	}

	var dobTime time.Time
	var err error

	dobTime, err = time.Parse(time.RFC3339, *user.DOB)
	if err != nil {
		dobTime, err = time.Parse("2006-01-02", *user.DOB)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid date of birth format. Expected YYYY-MM-DD, got: %s", *user.DOB)
		}
	}

	now := time.Now()
	age := now.Year() - dobTime.Year()
	if now.YearDay() < dobTime.YearDay() {
		age--
	}

	if profile.MacrosomicBaby == nil {
		return nil, nil, fmt.Errorf("macrosomic baby history is required but not found")
	}
	isMacrosomicBaby := *profile.MacrosomicBaby

	if profile.Bloodline == nil {
		return nil, nil, fmt.Errorf("bloodline status is required but not found")
	}
	isBloodline := *profile.Bloodline

	var (
		smokingStatus             int
		bmi                       float64
		isHypertension            bool
		physicalActivityFrequency int
		isCholesterol             bool
		brinkmanIndex             int
		avgSmokeCount             int
	)

	if input == nil {
		if profile.BMI == nil {
			return nil, nil, fmt.Errorf("BMI is required but not found")
		}
		bmi = *profile.BMI

		if profile.Hypertension == nil {
			return nil, nil, fmt.Errorf("hypertension status is required but not found")
		}
		isHypertension = *profile.Hypertension

		if profile.Cholesterol == nil {
			return nil, nil, fmt.Errorf("cholesterol status is required but not found")
		}
		isCholesterol = *profile.Cholesterol

		smokingStatus, err = w.calculateSmokingStatus(userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate smoking status: %v", err)
		}

		physicalActivityFrequency, err = w.calculatePhysicalActivityFrequency(userID, profile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate physical activity: %v", err)
		}

		if profile.SmokeCount != nil {
			avgSmokeCount = *profile.SmokeCount
			brinkmanIndex, err = w.calculateBrinkmanIndex(user, profile, avgSmokeCount)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to calculate Brinkman index: %v", err)
			}
		} else {
			avgSmokeCount = 0
			brinkmanIndex = 0
		}

	} else {
		smokingStatus = input.SmokingStatus
		bmi = float64(input.Weight) / math.Pow(float64(*profile.Height)/100, 2)
		isHypertension = input.IsHypertension
		physicalActivityFrequency = input.PhysicalActivityFrequency
		isCholesterol = input.IsCholesterol
		avgSmokeCount = input.AvgSmokeCount

		brinkmanIndex, err = w.calculateBrinkmanIndex(user, profile, input.AvgSmokeCount)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate Brinkman index: %v", err)
		}
	}

	features := []float64{
		float64(age),
		float64(smokingStatus),
		w.boolToFloat(isCholesterol),
		float64(isMacrosomicBaby),
		float64(physicalActivityFrequency),
		w.boolToFloat(isBloodline),
		float64(brinkmanIndex),
		bmi,
		w.boolToFloat(isHypertension),
	}

	featureInfo := map[string]interface{}{
		"age":                         age,
		"smoking_status":              smokingStatus,
		"is_macrosomic_baby":          isMacrosomicBaby,
		"brinkman_score":              brinkmanIndex,
		"bmi":                         bmi,
		"is_hypertension":             isHypertension,
		"is_cholesterol":              isCholesterol,
		"is_bloodline":                isBloodline,
		"physical_activity_frequency": physicalActivityFrequency,
		"avg_smoke_count":             avgSmokeCount,
	}

	return features, featureInfo, nil
}

func (w *PredictionJobWorker) calculateSmokingStatus(userID uint) (int, error) {
	profile, err := w.profileRepo.FindByUserID(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve user profile: %v", err)
	}

	user, err := w.userRepo.GetUserByID(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve user data: %v", err)
	}

	var currentAge int
	if user.DOB != nil && *user.DOB != "" {
		var dobTime time.Time
		var err error

		formats := []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}

		for _, format := range formats {
			dobTime, err = time.Parse(format, *user.DOB)
			if err == nil {
				break
			}
		}

		if err != nil {
			return 0, fmt.Errorf("failed to parse DOB: %v", err)
		}

		now := time.Now()
		currentAge = now.Year() - dobTime.Year()
		if now.Month() < dobTime.Month() || (now.Month() == dobTime.Month() && now.Day() < dobTime.Day()) {
			currentAge--
		}
	} else {
		return 0, fmt.Errorf("user DOB is required for age calculation")
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -56)

	recentActivities, err := w.activityRepo.GetActivitiesByUserIDAndTypeAndDateRange(userID, "smoke", startDate, endDate)
	if err != nil {
		return 0, err
	}

	if (profile.AgeOfSmoking == nil || *profile.AgeOfSmoking == 0) && len(recentActivities) == 0 {
		return 0, nil
	}

	if profile.AgeOfSmoking != nil && *profile.AgeOfSmoking != 0 && (profile.AgeOfStopSmoking == nil || *profile.AgeOfStopSmoking == 0) {
		return 2, nil
	}

	if len(recentActivities) > 0 {
		return 2, nil
	}

	if profile.AgeOfSmoking != nil && *profile.AgeOfSmoking != 0 &&
		profile.AgeOfStopSmoking != nil && *profile.AgeOfStopSmoking != 0 &&
		currentAge > *profile.AgeOfStopSmoking {
		return 1, nil
	}

	return 0, nil
}

func (w *PredictionJobWorker) calculateBrinkmanIndex(user *models.User, profile *models.UserProfile, avgSmokeCount int) (int, error) {
	now := time.Now()

	ageOfSmoking := 0
	if profile.AgeOfSmoking != nil {
		ageOfSmoking = *profile.AgeOfSmoking
	}

	yearsOfSmoking := 0

	if profile.AgeOfStopSmoking != nil {
		yearsOfSmoking = *profile.AgeOfStopSmoking - ageOfSmoking
	} else {
		if user.DOB == nil {
			return 0, fmt.Errorf("date of birth is required")
		}

		dob, err := time.Parse("2006-01-02", *user.DOB)
		if err != nil {
			return 0, fmt.Errorf("invalid date of birth format: %v", err)
		}

		age := now.Year() - dob.Year()
		if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
			age--
		}

		yearsOfSmoking = age - ageOfSmoking
	}

	if yearsOfSmoking < 0 {
		yearsOfSmoking = 0
	}

	brinkmanIndex := yearsOfSmoking * avgSmokeCount

	switch {
	case brinkmanIndex <= 0:
		return 0, nil
	case brinkmanIndex < 200:
		return 1, nil
	case brinkmanIndex < 600:
		return 2, nil
	default:
		return 3, nil
	}
}

func (w *PredictionJobWorker) calculatePhysicalActivityFrequency(userID uint, profile *models.UserProfile) (int, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	var totalFrequency int

	if profile.CreatedAt.Before(startDate) {
		activities, err := w.activityRepo.GetActivitiesByUserIDAndTypeAndDateRange(userID, "workout", startDate, endDate)
		if err != nil {
			return 0, err
		}

		totalFrequency = 0
		for _, activity := range activities {
			totalFrequency += activity.Value
		}
	} else if profile.PhysicalActivityFrequency != nil {
		totalFrequency = *profile.PhysicalActivityFrequency
	}

	return totalFrequency, nil
}

// ========== HELPER UTILITIES ==========

func (w *PredictionJobWorker) boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func (w *PredictionJobWorker) GetStatus() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	w.pendingJobsMu.RLock()
	pendingCount := len(w.pendingJobs)
	w.pendingJobsMu.RUnlock()

	return map[string]interface{}{
		"running":            w.running,
		"worker_count":       w.workerCount,
		"queue_size":         len(w.jobQueue),
		"queue_capacity":     cap(w.jobQueue),
		"pending_jobs":       pendingCount,
		"max_job_timeout":    w.maxJobTimeout.String(),
		"max_concurrency":    w.maxConcurrency,
		"cleanup_interval":   w.cleanupInterval.String(),
		"rabbitmq_connected": w.conn != nil && !w.conn.IsClosed(),
	}
}
