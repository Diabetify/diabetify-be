package services

import (
	"context"
	"diabetify/internal/cache"
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

// PredictionJobWorker defines the interface for prediction job processing
type PredictionJobWorker interface {
	// Lifecycle management
	Start()
	Stop()

	// Job submission
	SubmitJob(jobRequest models.PredictionJobRequest) error

	// Status and monitoring
	GetStatus() map[string]interface{}

	// What-if result handling
	GetWhatIfResult(jobID string) (map[string]interface{}, bool, error)
}

// predictionJobWorker is the concrete implementation
type predictionJobWorker struct {
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

	// RabbitMQ for ML responses (separate handler)
	conn            *amqp.Connection
	responseChannel *amqp.Channel

	// Configuration
	maxJobTimeout   time.Duration
	cleanupInterval time.Duration
	redisClient     *cache.RedisClient
}

// NewPredictionJobWorker creates a new prediction job worker
func NewPredictionJobWorker(
	jobRepo repository.PredictionJobRepository,
	predRepo repository.PredictionRepository,
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	activityRepo repository.ActivityRepository,
	mlClient ml.MLClient,
	workerCount int,
) PredictionJobWorker {
	if workerCount <= 0 {
		workerCount = 3
	}
	redisClient, err := cache.NewRedisClient()
	if err != nil {
		// Log error but don't fail - fallback to no caching
		fmt.Printf("Warning: Failed to connect to Redis: %v\n", err)
	}

	return &predictionJobWorker{
		jobRepo:         jobRepo,
		predRepo:        predRepo,
		userRepo:        userRepo,
		profileRepo:     profileRepo,
		activityRepo:    activityRepo,
		mlClient:        mlClient,
		jobQueue:        make(chan models.PredictionJobRequest, 2000),
		workerCount:     workerCount,
		stopChan:        make(chan struct{}),
		maxJobTimeout:   30 * time.Second,
		cleanupInterval: 30 * time.Minute,
		redisClient:     redisClient,
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

// ========== INTERFACE IMPLEMENTATIONS ==========

func (w *predictionJobWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	// Setup RabbitMQ response handler (separate from job processing)
	_ = w.setupRabbitMQResponseHandler()

	// Start worker goroutines for job processing
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}
	// Start job recovery routine
	w.wg.Add(1)
	go w.recoverPendingJobs()

	// Start cleanup routine
	w.wg.Add(1)
	go w.cleanupRoutine()
}

func (w *predictionJobWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	if w.redisClient != nil {
		w.redisClient.Close()
	}

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

func (w *predictionJobWorker) SubmitJob(jobRequest models.PredictionJobRequest) error {
	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return fmt.Errorf("job worker is not running")
	}
	w.mu.RUnlock()

	select {
	case w.jobQueue <- jobRequest:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("job queue is full, try again later")
	}
}

func (w *predictionJobWorker) GetStatus() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return map[string]interface{}{
		"running":            w.running,
		"worker_count":       w.workerCount,
		"queue_size":         len(w.jobQueue),
		"queue_capacity":     cap(w.jobQueue),
		"max_job_timeout":    w.maxJobTimeout.String(),
		"cleanup_interval":   w.cleanupInterval.String(),
		"rabbitmq_connected": w.conn != nil && !w.conn.IsClosed(),
		"pattern":            "fire_and_forget",
	}
}

func (w *predictionJobWorker) GetWhatIfResult(jobID string) (map[string]interface{}, bool, error) {
	if w.redisClient == nil {
		return nil, false, fmt.Errorf("Redis client not available")
	}

	return w.redisClient.GetWhatIfResult(jobID)
}

// ========== PRIVATE IMPLEMENTATION METHODS ==========

func (w *predictionJobWorker) setupRabbitMQResponseHandler() error {
	var err error
	w.conn, err = amqp.Dial("amqp://admin:password123@localhost:5672/")
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	w.responseChannel, err = w.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}

	_, err = w.responseChannel.QueueDeclare(
		"ml.prediction.hybrid_response", true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	msgs, err := w.responseChannel.Consume(
		"ml.prediction.hybrid_response", "response_handler", false, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %v", err)
	}

	w.wg.Add(1)
	go w.handleMLResponses(msgs)

	return nil
}

func (w *predictionJobWorker) handleMLResponses(msgs <-chan amqp.Delivery) {
	defer w.wg.Done()
	for {
		select {
		case <-w.stopChan:
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			var rabbitResponse RabbitMQPredictionResponse
			if err := json.Unmarshal(msg.Body, &rabbitResponse); err != nil {
				fmt.Printf("ERROR: Failed to unmarshal RabbitMQ message for CorrelationID %s: %v\n", msg.CorrelationId, err)
				msg.Nack(false, false)
				continue
			}
			w.handleSingleMLResponse(&rabbitResponse)
			_ = msg.Ack(false)
		}
	}
}

func (w *predictionJobWorker) handleSingleMLResponse(rabbitResponse *RabbitMQPredictionResponse) {
	jobID := rabbitResponse.CorrelationID

	job, err := w.jobRepo.GetJobByID(jobID)
	if err != nil {
		return
	}

	if job.Status != "submitted" {
		return
	}

	if rabbitResponse.Error != nil {
		errMsg := *rabbitResponse.Error
		_ = w.jobRepo.UpdateJobStatus(jobID, "failed", &errMsg)
		return
	}

	modelResponse := convertToModelsResponse(rabbitResponse)

	if w.isWhatIfJob(jobID) {
		featureInfo := w.extractFeatureInfoFromMLResponse(rabbitResponse, 0)
		whatIfResult := map[string]interface{}{
			"job_id":               jobID,
			"job_type":             "what_if",
			"risk_score":           modelResponse.Prediction,
			"risk_percentage":      modelResponse.Prediction * 100,
			"user_data_used":       featureInfo,
			"feature_explanations": w.buildFeatureExplanations(modelResponse),
			"timestamp":            time.Now(),
			"processing_time":      time.Since(job.CreatedAt).String(),
		}
		if err := w.storeWhatIfResult(jobID, whatIfResult); err != nil {
			fmt.Printf("Warning: Failed to store what-if result in Redis: %v\n", err)
		}
		_ = w.jobRepo.UpdateJobStatus(jobID, "completed", nil)
		return
	}

	// ===== REGULAR PREDICTION - SAVE TO DATABASE =====
	var avgSmokeCount int
	var calcErr error
	avgSmokeCount, calcErr = w.getAverageUserSmokeCount(job.UserID)
	if calcErr != nil {
		fmt.Printf("Warning: failed to calculate average smoke count for user %d: %v. Defaulting to 0.\n", job.UserID, calcErr)
		avgSmokeCount = 0
	}

	featureInfo := w.extractFeatureInfoFromMLResponse(rabbitResponse, avgSmokeCount)

	prediction := w.createPredictionRecord(job.UserID, modelResponse, featureInfo)

	if err := w.predRepo.SavePrediction(prediction); err != nil {
		errMsg := fmt.Sprintf("Failed to save prediction: %v", err)
		_ = w.jobRepo.UpdateJobStatus(jobID, "failed", &errMsg)
		return
	}

	now := time.Now()
	_ = w.userRepo.UpdateLastPredictionTime(job.UserID, &now)

	_ = w.jobRepo.UpdateJobStatusWithResult(jobID, "completed", prediction.ID)
}

func (w *predictionJobWorker) worker(workerID int) {
	defer w.wg.Done()
	for {
		select {
		case <-w.stopChan:
			return
		case jobRequest := <-w.jobQueue:
			w.processJobFireAndForget(jobRequest)
		}
	}
}

func (w *predictionJobWorker) processJobFireAndForget(jobRequest models.PredictionJobRequest) {
	jobID := jobRequest.JobID
	userID := jobRequest.UserID

	ctx, cancel := context.WithTimeout(context.Background(), w.maxJobTimeout)
	defer cancel()

	user, err := w.userRepo.GetUserByID(userID)
	if err != nil {
		errMsg := fmt.Sprintf("User not found: %v", err)
		_ = w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	profile, err := w.profileRepo.FindByUserID(userID)
	if err != nil {
		errMsg := fmt.Sprintf("Profile not found: %v", err)
		_ = w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	features, _, err := w.calculateFeaturesFromProfile(user, profile, userID, jobRequest.WhatIfInput)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to calculate features: %v", err)
		_ = w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	if err := w.jobRepo.UpdateJobStatus(jobID, models.JobStatusProcessing, nil); err != nil {
		return
	}

	correlationID := jobID
	if err := w.mlClient.PredictAsync(ctx, correlationID, features); err != nil {
		errMsg := fmt.Sprintf("Failed to submit to ML service: %v", err)
		_ = w.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)
		return
	}

	_ = w.jobRepo.UpdateJobStatus(jobID, models.JobStatusSubmitted, nil)
}

func (w *predictionJobWorker) recoverPendingJobs() {
	defer w.wg.Done()
	time.Sleep(5 * time.Second)
	pendingJobs, err := w.jobRepo.GetPendingJobs(50)
	if err != nil {
		return
	}
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

func (w *predictionJobWorker) cleanupRoutine() {
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

// ========== HELPER METHODS ==========

func parseTimestamp(timestampStr string) time.Time {
	formats := []string{
		"2006-01-02T15:04:05.000000", "2006-01-02T15:04:05", time.RFC3339, time.RFC3339Nano,
		"2006-01-02T15:04:05Z", "2006-01-02T15:04:05.000Z", "2006-01-02T15:04:05.000000Z", "2006-01-02 15:04:05",
	}
	for _, format := range formats {
		if parsedTime, err := time.Parse(format, timestampStr); err == nil {
			return parsedTime
		}
	}
	return time.Now()
}

func convertToModelsResponse(rabbitResponse *RabbitMQPredictionResponse) *models.PredictionResponse {
	explanation := make(map[string]models.ExplanationItem)
	for featureName, featureData := range rabbitResponse.Explanation {
		item := models.ExplanationItem{}
		if shap, ok := featureData["shap"].(float64); ok {
			item.Shap = shap
		}
		if contribution, ok := featureData["contribution"].(float64); ok {
			item.Contribution = contribution
		}
		if impact, ok := featureData["impact"].(float64); ok {
			item.Impact = int(impact)
		}
		explanation[featureName] = item
	}
	return &models.PredictionResponse{
		Prediction:  rabbitResponse.Prediction,
		Explanation: explanation,
		ElapsedTime: rabbitResponse.ElapsedTime,
		Timestamp:   parseTimestamp(rabbitResponse.Timestamp),
	}
}

func (w *predictionJobWorker) buildFeatureExplanations(response *models.PredictionResponse) map[string]map[string]interface{} {
	explanations := make(map[string]map[string]interface{})
	features := []struct{ name, key string }{
		{"age", "age"}, {"BMI", "BMI"}, {"brinkman_index", "brinkman_index"},
		{"is_hypertension", "is_hypertension"}, {"is_cholesterol", "is_cholesterol"},
		{"is_bloodline", "is_bloodline"}, {"is_macrosomic_baby", "is_macrosomic_baby"},
		{"smoking_status", "smoking_status"}, {"moderate_physical_activity_frequency", "moderate_physical_activity_frequency"},
	}
	for _, f := range features {
		if exp, ok := response.Explanation[f.key]; ok {
			explanations[f.name] = map[string]interface{}{"shap": exp.Shap, "contribution": exp.Contribution, "impact": exp.Impact}
		}
	}
	return explanations
}

func (w *predictionJobWorker) isWhatIfJob(jobID string) bool {
	job, err := w.jobRepo.GetJobByID(jobID)
	if err != nil {
		return false
	}
	return job.IsWhatIf
}

func (w *predictionJobWorker) extractFeatureInfoFromMLResponse(response *RabbitMQPredictionResponse, avgSmokeCount int) map[string]interface{} {
	featureInfo := make(map[string]interface{})

	for featureName, featureData := range response.Explanation {
		if value, exists := featureData["value"]; exists && value != nil {
			switch featureName {
			case "age":
				if v, ok := value.(float64); ok {
					featureInfo["age"] = int(v)
				}
			case "smoking_status":
				if v, ok := value.(float64); ok {
					featureInfo["smoking_status"] = int(v)
				}
			case "is_cholesterol":
				if v, ok := value.(float64); ok {
					featureInfo["is_cholesterol"] = v == 1.0
				}
			case "is_macrosomic_baby":
				if v, ok := value.(float64); ok {
					featureInfo["is_macrosomic_baby"] = int(v)
				}
			case "moderate_physical_activity_frequency":
				if v, ok := value.(float64); ok {
					featureInfo["physical_activity_frequency"] = int(v)
				}
			case "is_bloodline":
				if v, ok := value.(float64); ok {
					featureInfo["is_bloodline"] = v == 1.0
				}
			case "brinkman_index":
				if v, ok := value.(float64); ok {
					featureInfo["brinkman_score"] = int(v)
				}
			case "BMI":
				if v, ok := value.(float64); ok {
					featureInfo["bmi"] = v
				}
			case "is_hypertension":
				if v, ok := value.(float64); ok {
					featureInfo["is_hypertension"] = v == 1.0
				}
			}
		}
	}

	// The ML model does not know about avg_smoke_count, so it's never in the response.
	// We must add it here manually from the value we calculated.
	featureInfo["avg_smoke_count"] = avgSmokeCount

	return featureInfo
}

func (w *predictionJobWorker) createPredictionRecord(userID uint, response *models.PredictionResponse, featureInfo map[string]interface{}) *models.Prediction {
	getExplanation := func(key string) (float64, float64, float64) {
		if exp, exists := response.Explanation[key]; exists {
			return exp.Shap, exp.Contribution, float64(exp.Impact)
		}
		return 0.0, 0.0, 0.0
	}

	getInt := func(key string) int {
		if val, ok := featureInfo[key].(int); ok {
			return val
		}
		if val, ok := featureInfo[key].(float64); ok {
			return int(val)
		}
		return 0
	}

	getFloat := func(key string) float64 {
		if val, ok := featureInfo[key].(float64); ok {
			return val
		}
		if val, ok := featureInfo[key].(int); ok {
			return float64(val)
		}
		return 0.0
	}

	getBool := func(key string) bool {
		if val, ok := featureInfo[key].(bool); ok {
			return val
		}
		return false
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

		Age:             getInt("age"),
		AgeShap:         ageShap,
		AgeContribution: ageContribution,
		AgeImpact:       ageImpact,

		BMI:             getFloat("bmi"),
		BMIShap:         bmiShap,
		BMIContribution: bmiContribution,
		BMIImpact:       bmiImpact,

		BrinkmanScore:             getInt("brinkman_score"),
		BrinkmanScoreShap:         brinkmanShap,
		BrinkmanScoreContribution: brinkmanContribution,
		BrinkmanScoreImpact:       brinkmanImpact,

		IsHypertension:             getBool("is_hypertension"),
		IsHypertensionShap:         hypertensionShap,
		IsHypertensionContribution: hypertensionContribution,
		IsHypertensionImpact:       hypertensionImpact,

		IsCholesterol:             getBool("is_cholesterol"),
		IsCholesterolShap:         cholesterolShap,
		IsCholesterolContribution: cholesterolContribution,
		IsCholesterolImpact:       cholesterolImpact,

		IsBloodline:             getBool("is_bloodline"),
		IsBloodlineShap:         bloodlineShap,
		IsBloodlineContribution: bloodlineContribution,
		IsBloodlineImpact:       bloodlineImpact,

		IsMacrosomicBaby:             getInt("is_macrosomic_baby"),
		IsMacrosomicBabyShap:         macrosomicShap,
		IsMacrosomicBabyContribution: macrosomicContribution,
		IsMacrosomicBabyImpact:       macrosomicImpact,

		SmokingStatus:             getInt("smoking_status"),
		SmokingStatusShap:         smokingShap,
		SmokingStatusContribution: smokingContribution,
		SmokingStatusImpact:       smokingImpact,

		PhysicalActivityFrequency:             getInt("physical_activity_frequency"),
		PhysicalActivityFrequencyShap:         activityShap,
		PhysicalActivityFrequencyContribution: activityContribution,
		PhysicalActivityFrequencyImpact:       activityImpact,

		AvgSmokeCount: getInt("avg_smoke_count"),
	}
}

func (w *predictionJobWorker) storeWhatIfResult(jobID string, result map[string]interface{}) error {
	if w.redisClient == nil {
		return fmt.Errorf("Redis client not available")
	}
	return w.redisClient.StoreWhatIfResult(jobID, result, 1*time.Hour)
}

// ========== FEATURE CALCULATION METHODS ==========

func (w *predictionJobWorker) getAverageUserSmokeCount(userID uint) (int, error) {
	profile, err := w.profileRepo.FindByUserID(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get profile for user %d for smoke count: %v", userID, err)
	}

	// --- NEW LOGIC: Prioritize self-reported smoke count ---
	if profile != nil && profile.SmokeCount != nil && *profile.SmokeCount > 0 {
		return *profile.SmokeCount, nil
	}

	// Fallback to activity-based calculation only if profile.SmokeCount is not available.
	user, err := w.userRepo.GetUserByID(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user %d for smoke count: %v", userID, err)
	}
	activities, err := w.activityRepo.GetActivitiesByUserIDAndType(userID, "smoke")
	if err != nil {
		return 0, fmt.Errorf("failed to get smoke activities for user %d: %v", userID, err)
	}

	if len(activities) == 0 {
		return 0, nil
	}

	totalSmoked := 0
	for _, activity := range activities {
		totalSmoked += activity.Value
	}

	if user != nil && user.DOB != nil && profile != nil && profile.AgeOfSmoking != nil {
		var dobTime time.Time
		var parseErr error
		dobTime, parseErr = time.Parse(time.RFC3339, *user.DOB)
		if parseErr != nil {
			dobTime, parseErr = time.Parse("2006-01-02", *user.DOB)
		}
		if parseErr == nil {
			now := time.Now()
			age := now.Year() - dobTime.Year()
			if now.YearDay() < dobTime.YearDay() {
				age--
			}
			ageOfStartSmoking := *profile.AgeOfSmoking
			if age > ageOfStartSmoking {
				startSmokingDate := dobTime.AddDate(ageOfStartSmoking, 0, 0)
				durationDays := time.Since(startSmokingDate).Hours() / 24
				if durationDays >= 1 {
					average := float64(totalSmoked) / durationDays
					return int(math.Ceil(average)), nil
				}
			}
		}
	}

	if len(activities) == 1 {
		return activities[0].Value, nil
	}
	firstDate := activities[0].ActivityDate
	lastDate := activities[0].ActivityDate
	for _, activity := range activities {
		if activity.ActivityDate.Before(firstDate) {
			firstDate = activity.ActivityDate
		}
		if activity.ActivityDate.After(lastDate) {
			lastDate = activity.ActivityDate
		}
	}
	durationDays := int(lastDate.Sub(firstDate).Hours()/24) + 1
	if durationDays <= 0 {
		durationDays = 1
	}
	average := float64(totalSmoked) / float64(durationDays)
	return int(math.Ceil(average)), nil
}

func (w *predictionJobWorker) calculateFeaturesFromProfile(user *models.User, profile *models.UserProfile, userID uint, input *models.WhatIfInput) ([]float64, map[string]interface{}, error) {
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

		avgSmokeCount, err = w.getAverageUserSmokeCount(userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate average smoke count: %v", err)
		}

		brinkmanIndex, err = w.calculateBrinkmanIndex(user, profile, avgSmokeCount)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate Brinkman index: %v", err)
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

func (w *predictionJobWorker) calculateSmokingStatus(userID uint) (int, error) {
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
		formats := []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}
		for _, format := range formats {
			if dobTime, err = time.Parse(format, *user.DOB); err == nil {
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

func (w *predictionJobWorker) calculateBrinkmanIndex(user *models.User, profile *models.UserProfile, avgSmokeCount int) (int, error) {
	now := time.Now()
	ageOfSmoking := 0
	if profile.AgeOfSmoking != nil {
		ageOfSmoking = *profile.AgeOfSmoking
	}
	yearsOfSmoking := 0
	if profile.AgeOfStopSmoking != nil && *profile.AgeOfStopSmoking != 0 {
		// Case 1: User has stopped smoking. Duration is fixed.
		yearsOfSmoking = *profile.AgeOfStopSmoking - ageOfSmoking
	} else {
		// Case 2: User is still smoking. Calculate duration up to their current age.
		if user.DOB == nil {
			return 0, fmt.Errorf("date of birth is required")
		}
		dob, err := time.Parse(time.RFC3339, *user.DOB)
		if err != nil {
			dob, err = time.Parse("2006-01-02", *user.DOB)
			if err != nil {
				return 0, fmt.Errorf("invalid date of birth format: %v", err)
			}
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
	rawBrinkmanIndex := yearsOfSmoking * avgSmokeCount
	var categorizedIndex int
	switch {
	case rawBrinkmanIndex <= 0:
		categorizedIndex = 0
	case rawBrinkmanIndex < 200:
		categorizedIndex = 1
	case rawBrinkmanIndex < 600:
		categorizedIndex = 2
	default:
		categorizedIndex = 3
	}
	return categorizedIndex, nil
}

func (w *predictionJobWorker) calculatePhysicalActivityFrequency(userID uint, profile *models.UserProfile) (int, error) {
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

func (w *predictionJobWorker) boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
