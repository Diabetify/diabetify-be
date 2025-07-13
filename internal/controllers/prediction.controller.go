package controllers

import (
	"context"
	"diabetify/internal/ml"
	"diabetify/internal/models"
	"diabetify/internal/openai"
	"diabetify/internal/repository"
	"diabetify/internal/services"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PredictionController struct {
	repo         repository.PredictionRepository
	userRepo     repository.UserRepository
	profileRepo  repository.UserProfileRepository
	activityRepo repository.ActivityRepository
	jobRepo      repository.PredictionJobRepository
	jobWorker    *services.PredictionJobWorker
	mlClient     ml.MLClient
}

func NewPredictionController(
	repo repository.PredictionRepository,
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	activityRepo repository.ActivityRepository,
	jobRepo repository.PredictionJobRepository,
	jobWorker *services.PredictionJobWorker,
	mlClient ml.MLClient,
) *PredictionController {
	return &PredictionController{
		repo:         repo,
		userRepo:     userRepo,
		profileRepo:  profileRepo,
		activityRepo: activityRepo,
		jobRepo:      jobRepo,
		jobWorker:    jobWorker,
		mlClient:     mlClient,
	}
}

// TestMLConnection godoc
// @Summary Test ML service connection
// @Description Test the connection to the async ML service via RabbitMQ
// @Tags prediction
// @Produce json
// @Success 200 {object} map[string]interface{} "ML service is healthy"
// @Failure 500 {object} map[string]interface{} "ML service is not reachable"
// @Router /prediction/health [get]
func (pc *PredictionController) TestMLConnection(c *gin.Context) {
	// Check if job worker is available and running
	if pc.jobWorker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Job worker is not available",
		})
		return
	}

	// Get job worker status
	status := pc.jobWorker.GetStatus()

	// Check if worker is running and RabbitMQ is connected
	isRunning, runningOk := status["running"].(bool)
	isRabbitMQConnected, rabbitOk := status["rabbitmq_connected"].(bool)

	if runningOk && isRunning && rabbitOk && isRabbitMQConnected {
		// Optionally test async health check
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var healthCheckStatus string = "not_tested"
		if pc.mlClient != nil {
			// Try async health check (non-blocking)
			if err := pc.mlClient.HealthCheckAsync(ctx); err == nil {
				healthCheckStatus = "message_sent"
			} else {
				healthCheckStatus = "failed_to_send"
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "success",
			"message":   "Async ML service is healthy via RabbitMQ",
			"timestamp": time.Now(),
			"details": gin.H{
				"worker_status": status,
				"communication": "rabbitmq",
				"health_check":  healthCheckStatus,
			},
		})
		return
	}

	// Service is not healthy
	c.JSON(http.StatusInternalServerError, gin.H{
		"status":  "error",
		"message": "Async ML service is not reachable via RabbitMQ",
		"details": gin.H{
			"service_type":  "async_only",
			"worker_status": status,
		},
	})
}

// MakePrediction godoc
// @Summary Make an asynchronous prediction using user's profile data
// @Description Submit a prediction job using RabbitMQ for asynchronous processing
// @Tags prediction
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 202 {object} map[string]interface{} "Prediction job submitted"
// @Failure 400 {object} map[string]interface{} "Incomplete user profile"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User profile not found"
// @Failure 500 {object} map[string]interface{} "Failed to submit job"
// @Router /prediction [post]
func (pc *PredictionController) MakePrediction(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	// Validate user profile exists and is complete
	if err := pc.validateUserProfile(userID.(uint)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Incomplete user profile",
			"error":   err.Error(),
			"help":    "Please ensure all required profile fields are filled: age, weight, height, smoking status, macrosomic baby history, hypertension status, cholesterol status, diabetes bloodline",
		})
		return
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Create job record in database
	job := &models.PredictionJob{
		ID:        jobID,
		UserID:    userID.(uint),
		Status:    models.JobStatusPending,
		IsWhatIf:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := pc.jobRepo.SaveJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create prediction job",
			"error":   err.Error(),
		})
		return
	}

	jobRequest := models.PredictionJobRequest{
		JobID:       jobID,
		UserID:      userID.(uint),
		WhatIfInput: nil, // Regular prediction
	}

	if err := pc.jobWorker.SubmitJob(jobRequest); err != nil {
		// Update job status to failed
		errMsg := fmt.Sprintf("Failed to submit job: %v", err)
		pc.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to submit prediction job",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "success",
		"message": "Prediction job submitted successfully",
		"data": gin.H{
			"job_id":      jobID,
			"status":      models.JobStatusPending,
			"submit_time": time.Now(),
			"poll_url":    fmt.Sprintf("/prediction/job/%s/status", jobID),
		},
	})
}

// WhatIfPrediction godoc
// @Summary Make an asynchronous what-if prediction
// @Description Submit a what-if prediction job using custom parameters via RabbitMQ
// @Tags prediction
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param input body models.WhatIfInput true "What-if prediction parameters"
// @Success 202 {object} map[string]interface{} "What-if prediction job submitted"
// @Failure 400 {object} map[string]interface{} "Invalid input or incomplete profile"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to submit job"
// @Router /prediction/what-if [post]
func (pc *PredictionController) WhatIfPrediction(c *gin.Context) {
	var input models.WhatIfInput
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid input format",
			"error":   err.Error(),
		})
		return
	}

	// Validate user profile exists (basic data needed for what-if)
	if err := pc.validateUserProfile(userID.(uint)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Incomplete user profile",
			"error":   err.Error(),
		})
		return
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Create job record in database
	job := &models.PredictionJob{
		ID:        jobID,
		UserID:    userID.(uint),
		Status:    models.JobStatusPending,
		IsWhatIf:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := pc.jobRepo.SaveJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create what-if prediction job",
			"error":   err.Error(),
		})
		return
	}

	jobRequest := models.PredictionJobRequest{
		JobID:       jobID,
		UserID:      userID.(uint),
		WhatIfInput: &input, // What-if prediction with custom parameters
	}

	if err := pc.jobWorker.SubmitJob(jobRequest); err != nil {
		// Update job status to failed
		errMsg := fmt.Sprintf("Failed to submit job: %v", err)
		pc.jobRepo.UpdateJobStatus(jobID, models.JobStatusFailed, &errMsg)

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to submit what-if prediction job",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "success",
		"message": "What-if prediction job submitted successfully",
		"data": gin.H{
			"job_id":      jobID,
			"status":      models.JobStatusPending,
			"submit_time": time.Now(),
			"poll_url":    fmt.Sprintf("/prediction/job/%s/status", jobID),
			"input_used":  input,
		},
	})
}

// GetJobStatus godoc
// @Summary Get prediction job status
// @Description Get the current status and progress of a prediction job
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Param job_id path string true "Job ID"
// @Success 200 {object} map[string]interface{} "Job status retrieved"
// @Failure 400 {object} map[string]interface{} "Invalid job ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Job belongs to different user"
// @Failure 404 {object} map[string]interface{} "Job not found"
// @Router /prediction/job/{job_id}/status [get]
func (pc *PredictionController) GetJobStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Job ID is required",
		})
		return
	}

	job, err := pc.jobRepo.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Job not found",
			"error":   err.Error(),
		})
		return
	}

	if job.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Job belongs to a different user",
		})
		return
	}

	response := gin.H{
		"status":  "success",
		"message": "Job status retrieved successfully",
		"data": gin.H{
			"job_id":     job.ID,
			"status":     job.Status, // Only status
			"created_at": job.CreatedAt,
			"updated_at": job.UpdatedAt,
		},
	}

	// Add status-specific information
	switch job.Status {
	case "pending":
		response["data"].(gin.H)["message"] = "Job is waiting to be processed"
	case "processing":
		response["data"].(gin.H)["message"] = "Job is being prepared for ML service"
	case "submitted":
		response["data"].(gin.H)["message"] = "Job has been submitted to ML service and is being processed"
		response["data"].(gin.H)["note"] = "This may take a few minutes. ML service will respond when ready."
	case "completed":
		response["data"].(gin.H)["message"] = "Job completed successfully"
		if job.PredictionID != nil {
			prediction, err := pc.repo.GetPredictionByID(*job.PredictionID)
			if err == nil {
				response["data"].(gin.H)["result"] = gin.H{
					"prediction_id":   prediction.ID,
					"risk_score":      prediction.RiskScore,
					"risk_percentage": prediction.RiskScore * 100,
					"created_at":      prediction.CreatedAt,
				}
			}
		}
	case "failed":
		response["data"].(gin.H)["message"] = "Job failed"
		if job.ErrorMessage != nil {
			response["data"].(gin.H)["error"] = *job.ErrorMessage
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetJobResult godoc
// @Summary Get prediction job result
// @Description Get the detailed result of a completed fire-and-forget prediction job
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Param job_id path string true "Job ID"
// @Success 200 {object} map[string]interface{} "Job result retrieved"
// @Failure 400 {object} map[string]interface{} "Invalid job ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Job belongs to different user"
// @Failure 404 {object} map[string]interface{} "Job not found or not completed"
// @Router /prediction/job/{job_id}/result [get]
func (pc *PredictionController) GetJobResult(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Job ID is required",
		})
		return
	}

	// Get job from database
	job, err := pc.jobRepo.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Job not found",
			"error":   err.Error(),
		})
		return
	}

	// Check if job belongs to user
	if job.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Job belongs to a different user",
		})
		return
	}

	// Check if job is completed
	if job.Status != models.JobStatusCompleted {
		c.JSON(http.StatusOK, gin.H{
			"status":  "succcess",
			"message": fmt.Sprintf("Job is not completed yet. Current status: %s", job.Status),
			"current_status": gin.H{
				"status": job.Status,
			},
		})
		return
	}
	if job.IsWhatIf {
		whatIfResult, exists, err := pc.jobWorker.GetWhatIfResult(jobID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to retrieve what-if result",
				"error":   err.Error(),
			})
			return
		}

		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "What-if result has expired or not found",
				"help":    "What-if results are only available for 1 hours after completion",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "What-if prediction result retrieved successfully",
			"data":    whatIfResult,
		})
		return

	} else {
		// ===== REGULAR PREDICTION FROM DATABASE =====
		if job.PredictionID == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "Job completed but no result found",
			})
			return
		}
	}
	// Check if result exists
	if job.PredictionID == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Job completed but no result found",
		})
		return
	}

	// Get prediction result
	prediction, err := pc.repo.GetPredictionByID(*job.PredictionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Prediction result not found",
			"error":   err.Error(),
		})
		return
	}

	// Build feature explanations map
	featureExplanations := make(map[string]map[string]interface{})
	features := []struct {
		name         string
		shap         float64
		contribution float64
		impact       float64
	}{
		{"age", prediction.AgeShap, prediction.AgeContribution, prediction.AgeImpact},
		{"BMI", prediction.BMIShap, prediction.BMIContribution, prediction.BMIImpact},
		{"brinkman_index", prediction.BrinkmanScoreShap, prediction.BrinkmanScoreContribution, prediction.BrinkmanScoreImpact},
		{"is_hypertension", prediction.IsHypertensionShap, prediction.IsHypertensionContribution, prediction.IsHypertensionImpact},
		{"is_cholesterol", prediction.IsCholesterolShap, prediction.IsCholesterolContribution, prediction.IsCholesterolImpact},
		{"is_bloodline", prediction.IsBloodlineShap, prediction.IsBloodlineContribution, prediction.IsBloodlineImpact},
		{"is_macrosomic_baby", prediction.IsMacrosomicBabyShap, prediction.IsMacrosomicBabyContribution, prediction.IsMacrosomicBabyImpact},
		{"smoking_status", prediction.SmokingStatusShap, prediction.SmokingStatusContribution, prediction.SmokingStatusImpact},
		{"moderate_physical_activity_frequency", prediction.PhysicalActivityFrequencyShap, prediction.PhysicalActivityFrequencyContribution, prediction.PhysicalActivityFrequencyImpact},
	}

	for _, feature := range features {
		featureExplanations[feature.name] = map[string]interface{}{
			"shap":         feature.shap,
			"contribution": feature.contribution,
			"impact":       int(feature.impact),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Job result retrieved successfully",
		"data": gin.H{
			"job_id":          job.ID,
			"prediction_id":   prediction.ID,
			"risk_score":      prediction.RiskScore,
			"risk_percentage": prediction.RiskScore * 100,
			"timestamp":       prediction.CreatedAt,
			"user_data_used": gin.H{
				"age":                         prediction.Age,
				"smoking_status":              prediction.SmokingStatus,
				"is_macrosomic_baby":          prediction.IsMacrosomicBaby,
				"brinkman_score":              prediction.BrinkmanScore,
				"bmi":                         prediction.BMI,
				"is_hypertension":             prediction.IsHypertension,
				"is_cholesterol":              prediction.IsCholesterol,
				"is_bloodline":                prediction.IsBloodline,
				"physical_activity_frequency": prediction.PhysicalActivityFrequency,
			},
			"feature_explanations": featureExplanations,
			"job_info": gin.H{
				"completed_at":    job.UpdatedAt,
				"processing_time": job.UpdatedAt.Sub(job.CreatedAt).String(),
			},
		},
	})
}

// GetUserJobs godoc
// @Summary Get user's prediction jobs
// @Description Get all prediction jobs for the authenticated user
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Param status query string false "Filter by job status (pending, processing, completed, failed)"
// @Param limit query int false "Limit number of jobs returned (default: 10)"
// @Success 200 {object} map[string]interface{} "Jobs retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve jobs"
// @Router /prediction/jobs [get]
func (pc *PredictionController) GetUserJobs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	// Parse query parameters
	status := c.Query("status")
	limitStr := c.Query("limit")
	limit := 10 // Default limit
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid limit parameter",
				"error":   "Limit must be a positive integer",
			})
			return
		}
	}

	// Get jobs from database
	var jobs []*models.PredictionJob
	var err error

	if status != "" {
		jobs, err = pc.jobRepo.GetJobsByUserIDAndStatus(userID.(uint), status, limit)
	} else {
		jobs, err = pc.jobRepo.GetJobsByUserID(userID.(uint), limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve jobs",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Jobs retrieved successfully",
		"data": gin.H{
			"jobs":  jobs,
			"count": len(jobs),
		},
	})
}

// CancelJob godoc
// @Summary Cancel a prediction job
// @Description Cancel a pending or processing prediction job
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Param job_id path string true "Job ID"
// @Success 200 {object} map[string]interface{} "Job cancelled successfully"
// @Failure 400 {object} map[string]interface{} "Invalid job ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Job belongs to different user"
// @Failure 404 {object} map[string]interface{} "Job not found"
// @Router /prediction/job/{job_id}/cancel [post]
func (pc *PredictionController) CancelJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Job ID is required",
		})
		return
	}

	// Get job to verify ownership
	job, err := pc.jobRepo.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Job not found",
			"error":   err.Error(),
		})
		return
	}

	// Check if job belongs to user
	if job.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Job belongs to a different user",
		})
		return
	}

	// Check if job can be cancelled
	if job.Status == models.JobStatusSubmitted {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Cannot cancel job that has been submitted to ML service",
			"note":    "Job is already being processed by ML service",
		})
		return
	}

	// Cancel the job
	if err := pc.jobRepo.CancelJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Failed to cancel job",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Job cancelled successfully",
		"data": gin.H{
			"job_id":       jobID,
			"cancelled_at": time.Now(),
		},
	})
}

// validateUserProfile checks if user profile has all required fields
func (pc *PredictionController) validateUserProfile(userID uint) error {
	// Get user data
	user, err := pc.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	// Get user profile
	profile, err := pc.profileRepo.FindByUserID(userID)
	if err != nil {
		return fmt.Errorf("user profile not found: %v", err)
	}

	// Validate required fields
	if user.DOB == nil {
		return fmt.Errorf("date of birth is required")
	}

	if profile.BMI == nil {
		return fmt.Errorf("BMI is required")
	}

	if profile.MacrosomicBaby == nil {
		return fmt.Errorf("macrosomic baby history is required")
	}

	if profile.Bloodline == nil {
		return fmt.Errorf("diabetes bloodline status is required")
	}

	if profile.Hypertension == nil {
		return fmt.Errorf("hypertension status is required")
	}

	if profile.Cholesterol == nil {
		return fmt.Errorf("cholesterol status is required")
	}

	return nil
}

// ========== EXISTING METHODS (unchanged for backward compatibility) ==========

// GetUserPredictions godoc
// @Summary Get user's prediction history
// @Description Retrieve prediction history for the authenticated user
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Prediction history retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve prediction history"
// @Router /prediction/me [get]
func (pc *PredictionController) GetUserPredictions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	// Get Limit Params
	limitStr := c.Query("limit")
	limit := 10 // Default limit
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid limit parameter",
				"error":   "Limit must be a positive integer",
			})
			return
		}
	}

	predictions, err := pc.repo.GetPredictionsByUserID(userID.(uint), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve prediction history",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction history retrieved successfully",
		"data":    predictions,
	})
}

// GetPredictionsByDateRange - unchanged
func (pc *PredictionController) GetPredictionsByDateRange(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid start date format",
			"error":   "Date must be in YYYY-MM-DD format",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid end date format",
			"error":   "Date must be in YYYY-MM-DD format",
		})
		return
	}

	endDate = endDate.Add(24 * time.Hour).Add(-time.Second)

	predictions, err := pc.repo.GetPredictionsByUserIDAndDateRange(userID.(uint), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve prediction history",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction history retrieved successfully",
		"data":    predictions,
	})
}

// GetPredictionByID - unchanged
func (pc *PredictionController) GetPredictionByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid prediction ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	prediction, err := pc.repo.GetPredictionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Prediction not found",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if prediction.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Access denied: prediction belongs to a different user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction retrieved successfully",
		"data":    prediction,
	})
}

// DeletePrediction - unchanged
func (pc *PredictionController) DeletePrediction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid prediction ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	prediction, err := pc.repo.GetPredictionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Prediction not found",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if prediction.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Access denied: prediction belongs to a different user",
		})
		return
	}

	if err := pc.repo.DeletePrediction(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete prediction",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction deleted successfully",
	})
}

// GetPredictionScoreByDate - unchanged
func (pc *PredictionController) GetPredictionScoreByDate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid start date format",
			"error":   "Date must be in YYYY-MM-DD format",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid end date format",
			"error":   "Date must be in YYYY-MM-DD format",
		})
		return
	}

	endDate = endDate.Add(24 * time.Hour).Add(-time.Second)

	scores, err := pc.repo.GetPredictionScoreByUserIDAndDateRange(userID.(uint), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve prediction score",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction score retrieved successfully",
		"data":    scores,
	})
}

// GetLatestPredictionExplanation - unchanged
func (pc *PredictionController) GetLatestPredictionExplanation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	prediction, err := pc.repo.GetLatestPredictionByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "No prediction found",
			"error":   err.Error(),
		})
		return
	}

	hasExplanations := prediction.AgeExplanation != "" &&
		prediction.BMIExplanation != "" &&
		prediction.BrinkmanScoreExplanation != "" &&
		prediction.IsHypertensionExplanation != "" &&
		prediction.IsCholesterolExplanation != "" &&
		prediction.IsBloodlineExplanation != "" &&
		prediction.IsMacrosomicBabyExplanation != "" &&
		prediction.SmokingStatusExplanation != "" &&
		prediction.PhysicalActivityFrequencyExplanation != ""

	if hasExplanations {
		factorExplanations := map[string]string{
			"age":                         prediction.AgeExplanation,
			"bmi":                         prediction.BMIExplanation,
			"brinkman_score":              prediction.BrinkmanScoreExplanation,
			"is_hypertension":             prediction.IsHypertensionExplanation,
			"is_cholesterol":              prediction.IsCholesterolExplanation,
			"is_bloodline":                prediction.IsBloodlineExplanation,
			"is_macrosomic_baby":          prediction.IsMacrosomicBabyExplanation,
			"smoking_status":              prediction.SmokingStatusExplanation,
			"physical_activity_frequency": prediction.PhysicalActivityFrequencyExplanation,
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Latest prediction explanation retrieved successfully",
			"data": gin.H{
				"explanations":       factorExplanations,
				"prediction_summary": prediction.PredictionSummary,
			},
		})
		return
	}

	if os.Getenv("OPENAI_API_KEY") == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "OpenAI API key is not configured",
		})
		return
	}

	openaiClient, err := openai.NewClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to initialize OpenAI client",
			"error":   err.Error(),
		})
		return
	}

	factors := map[string]struct {
		Value        string
		Shap         float64
		Contribution float64
		Impact       float64
	}{
		"age": {
			Value:        fmt.Sprintf("%d years", prediction.Age),
			Shap:         prediction.AgeShap,
			Contribution: prediction.AgeContribution,
			Impact:       prediction.AgeImpact,
		},
		"bmi": {
			Value:        fmt.Sprintf("%.1f", prediction.BMI),
			Shap:         prediction.BMIShap,
			Contribution: prediction.BMIContribution,
			Impact:       prediction.BMIImpact,
		},
		"brinkman_score": {
			Value:        fmt.Sprintf("%.1f", prediction.BrinkmanScore),
			Shap:         prediction.BrinkmanScoreShap,
			Contribution: prediction.BrinkmanScoreContribution,
			Impact:       prediction.BrinkmanScoreImpact,
		},
		"is_hypertension": {
			Value:        fmt.Sprintf("%v", prediction.IsHypertension),
			Shap:         prediction.IsHypertensionShap,
			Contribution: prediction.IsHypertensionContribution,
			Impact:       prediction.IsHypertensionImpact,
		},
		"is_cholesterol": {
			Value:        fmt.Sprintf("%v", prediction.IsCholesterol),
			Shap:         prediction.IsCholesterolShap,
			Contribution: prediction.IsCholesterolContribution,
			Impact:       prediction.IsCholesterolImpact,
		},
		"is_bloodline": {
			Value:        fmt.Sprintf("%v", prediction.IsBloodline),
			Shap:         prediction.IsBloodlineShap,
			Contribution: prediction.IsBloodlineContribution,
			Impact:       prediction.IsBloodlineImpact,
		},
		"is_macrosomic_baby": {
			Value:        fmt.Sprintf("%v", prediction.IsMacrosomicBaby),
			Shap:         prediction.IsMacrosomicBabyShap,
			Contribution: prediction.IsMacrosomicBabyContribution,
			Impact:       prediction.IsMacrosomicBabyImpact,
		},
		"smoking_status": {
			Value:        fmt.Sprintf("%v", prediction.SmokingStatus),
			Shap:         prediction.SmokingStatusShap,
			Contribution: prediction.SmokingStatusContribution,
			Impact:       prediction.SmokingStatusImpact,
		},
		"physical_activity_frequency": {
			Value:        fmt.Sprintf("%d times", prediction.PhysicalActivityFrequency),
			Shap:         prediction.PhysicalActivityFrequencyShap,
			Contribution: prediction.PhysicalActivityFrequencyContribution,
			Impact:       prediction.PhysicalActivityFrequencyImpact,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	explanations, summary, tokenUsage, err := openaiClient.GeneratePredictionExplanation(ctx, prediction.RiskScore, factors)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to generate LLM explanation",
			"error":   err.Error(),
		})
		return
	}

	if len(explanations) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "No explanations were generated",
		})
		return
	}

	prediction.AgeExplanation = explanations["age"].Explanation
	prediction.BMIExplanation = explanations["bmi"].Explanation
	prediction.BrinkmanScoreExplanation = explanations["brinkman_score"].Explanation
	prediction.IsHypertensionExplanation = explanations["is_hypertension"].Explanation
	prediction.IsCholesterolExplanation = explanations["is_cholesterol"].Explanation
	prediction.IsBloodlineExplanation = explanations["is_bloodline"].Explanation
	prediction.IsMacrosomicBabyExplanation = explanations["is_macrosomic_baby"].Explanation
	prediction.SmokingStatusExplanation = explanations["smoking_status"].Explanation
	prediction.PhysicalActivityFrequencyExplanation = explanations["physical_activity_frequency"].Explanation
	prediction.PredictionSummary = summary

	if err := pc.repo.UpdatePrediction(prediction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update prediction with explanations",
			"error":   err.Error(),
		})
		return
	}

	factorExplanations := make(map[string]string)
	for factor, exp := range explanations {
		factorExplanations[factor] = exp.Explanation
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Latest prediction explanation generated successfully",
		"data": gin.H{
			"explanations":       factorExplanations,
			"prediction_summary": summary,
			"token_usage": gin.H{
				"prompt_tokens":     tokenUsage.PromptTokens,
				"completion_tokens": tokenUsage.CompletionTokens,
				"total_tokens":      tokenUsage.TotalTokens,
			},
		},
	})
}
