package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"diabetify/internal/controllers"
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupPredictionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupPredictionControllerWithMocks() (*controllers.PredictionController, *mocks.MockPredictionRepository, *mocks.MockUserRepository, *mocks.MockUserProfileRepository, *mocks.MockActivityRepository, *mocks.MockPredictionJobRepository, *mocks.MockPredictionJobWorker, *mocks.MockMLClient) {
	mockPredRepo := new(mocks.MockPredictionRepository)
	mockUserRepo := new(mocks.MockUserRepository)
	mockProfileRepo := new(mocks.MockUserProfileRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockJobRepo := new(mocks.MockPredictionJobRepository)
	mockJobWorker := new(mocks.MockPredictionJobWorker)
	mockMLClient := new(mocks.MockMLClient)

	controller := controllers.NewPredictionController(
		mockPredRepo,
		mockUserRepo,
		mockProfileRepo,
		mockActivityRepo,
		mockJobRepo,
		mockJobWorker,
		mockMLClient,
	)

	return controller, mockPredRepo, mockUserRepo, mockProfileRepo, mockActivityRepo, mockJobRepo, mockJobWorker, mockMLClient
}

func addPredictionAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestNewPredictionController(t *testing.T) {
	controller, _, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
	assert.NotNil(t, controller)
}

func TestMakePrediction(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMocks     func(*mocks.MockPredictionRepository, *mocks.MockUserRepository, *mocks.MockUserProfileRepository, *mocks.MockActivityRepository, *mocks.MockPredictionJobRepository, *mocks.MockPredictionJobWorker, *mocks.MockMLClient)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful async prediction submission",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				// Mock user data
				dob := "2000-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				// Mock profile data with all required fields
				bmi := 25.0
				hypertension := false
				cholesterol := false
				macrosomicBaby := 0
				bloodline := false
				profile := &models.UserProfile{
					BMI:            &bmi,
					Hypertension:   &hypertension,
					Cholesterol:    &cholesterol,
					MacrosomicBaby: &macrosomicBaby,
					Bloodline:      &bloodline,
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)

				// Mock job creation - SUCCESS
				jobRepo.On("SaveJob", mock.AnythingOfType("*models.PredictionJob")).Return(nil)

				// Mock job worker - SUCCESS (this is key)
				jobWorker.On("SubmitJob", mock.AnythingOfType("models.PredictionJobRequest")).Return(nil)
			},
			expectedStatus: http.StatusAccepted,
			expectedMsg:    "Prediction job submitted successfully",
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				userRepo.On("GetUserByID", uint(999)).Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Incomplete user profile",
		},
		{
			name:   "user profile not found",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				dob := "2000-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)
				profileRepo.On("FindByUserID", uint(1)).Return(nil, errors.New("profile not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Incomplete user profile",
		},
		{
			name:   "incomplete profile data - missing BMI",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				dob := "2000-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				// Profile with missing BMI
				profile := &models.UserProfile{
					BMI: nil,
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Incomplete user profile",
		},
		{
			name:   "job submission failed",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				dob := "2000-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				bmi := 25.0
				hypertension := false
				cholesterol := false
				macrosomicBaby := 0
				bloodline := false
				profile := &models.UserProfile{
					BMI:            &bmi,
					Hypertension:   &hypertension,
					Cholesterol:    &cholesterol,
					MacrosomicBaby: &macrosomicBaby,
					Bloodline:      &bloodline,
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)

				// Mock job save success but worker submission failure
				jobRepo.On("SaveJob", mock.AnythingOfType("*models.PredictionJob")).Return(nil)
				jobWorker.On("SubmitJob", mock.AnythingOfType("models.PredictionJobRequest")).Return(errors.New("job queue unavailable"))
				jobRepo.On("UpdateJobStatus", mock.AnythingOfType("string"), models.JobStatusFailed, mock.AnythingOfType("*string")).Return(nil)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to submit prediction job",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, userRepo, profileRepo, activityRepo, jobRepo, jobWorker, mlClient := setupPredictionControllerWithMocks()
			tt.setupMocks(predRepo, userRepo, profileRepo, activityRepo, jobRepo, jobWorker, mlClient)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.POST("/prediction", controller.MakePrediction)

			req := httptest.NewRequest("POST", "/prediction", nil)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			profileRepo.AssertExpectations(t)
			activityRepo.AssertExpectations(t)
			jobRepo.AssertExpectations(t)
			jobWorker.AssertExpectations(t)
			mlClient.AssertExpectations(t)
		})
	}
}

func TestMakePredictionUnauthorized(t *testing.T) {
	controller, _, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
	router := setupPredictionTestRouter()
	router.POST("/prediction", controller.MakePrediction)

	req := httptest.NewRequest("POST", "/prediction", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized access", response["message"])
}

func TestWhatIfPrediction(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockPredictionRepository, *mocks.MockUserRepository, *mocks.MockUserProfileRepository, *mocks.MockActivityRepository, *mocks.MockPredictionJobRepository, *mocks.MockPredictionJobWorker, *mocks.MockMLClient)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful what-if prediction",
			userID: 1,
			requestBody: map[string]interface{}{
				"smoking_status":              1,
				"avg_smoke_count":             5,
				"weight":                      70.0,
				"is_hypertension":             false,
				"physical_activity_frequency": 3,
				"is_cholesterol":              false,
			},
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				// Mock user data with DOB
				dob := "2000-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				// Mock profile data with ALL required fields for validation
				bmi := 25.0
				hypertension := false
				cholesterol := false
				macrosomicBaby := 0
				bloodline := false
				height := 175
				profile := &models.UserProfile{
					BMI:            &bmi,            // Required
					Hypertension:   &hypertension,   // Required
					Cholesterol:    &cholesterol,    // Required
					MacrosomicBaby: &macrosomicBaby, // Required
					Bloodline:      &bloodline,      // Required
					Height:         &height,         // Additional field
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)

				// Mock job creation and submission - SUCCESS
				jobRepo.On("SaveJob", mock.AnythingOfType("*models.PredictionJob")).Return(nil)
				jobWorker.On("SubmitJob", mock.AnythingOfType("models.PredictionJobRequest")).Return(nil)
			},
			expectedStatus: http.StatusAccepted,
			expectedMsg:    "What-if prediction job submitted successfully",
		},
		{
			name:   "invalid input format",
			userID: 1,
			requestBody: map[string]interface{}{
				"smoking_status": "invalid",
			},
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid input format",
		},
		{
			name:   "incomplete profile - missing required fields",
			userID: 1,
			requestBody: map[string]interface{}{
				"smoking_status":              1,
				"avg_smoke_count":             5,
				"weight":                      70.0,
				"is_hypertension":             false,
				"physical_activity_frequency": 3,
				"is_cholesterol":              false,
			},
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				dob := "2000-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				// Profile missing required BMI field
				macrosomicBaby := 0
				bloodline := false
				height := 175
				profile := &models.UserProfile{
					BMI:            nil, // Missing required field
					MacrosomicBaby: &macrosomicBaby,
					Bloodline:      &bloodline,
					Height:         &height,
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Incomplete user profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, userRepo, profileRepo, activityRepo, jobRepo, jobWorker, mlClient := setupPredictionControllerWithMocks()
			tt.setupMocks(predRepo, userRepo, profileRepo, activityRepo, jobRepo, jobWorker, mlClient)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.POST("/prediction/what-if", controller.WhatIfPrediction)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/prediction/what-if", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			profileRepo.AssertExpectations(t)
			activityRepo.AssertExpectations(t)
			jobRepo.AssertExpectations(t)
			jobWorker.AssertExpectations(t)
			mlClient.AssertExpectations(t)
		})
	}
}

func TestTestMLConnection(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockPredictionJobWorker, *mocks.MockMLClient)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "ML service healthy",
			setupMock: func(jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				status := map[string]interface{}{
					"running":            true,
					"rabbitmq_connected": true,
				}
				jobWorker.On("GetStatus").Return(status)
				mlClient.On("HealthCheckAsync", mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Async ML service is healthy via RabbitMQ",
		},
		{
			name: "ML service unhealthy - worker not running",
			setupMock: func(jobWorker *mocks.MockPredictionJobWorker, mlClient *mocks.MockMLClient) {
				status := map[string]interface{}{
					"running":            false,
					"rabbitmq_connected": false,
				}
				jobWorker.On("GetStatus").Return(status)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Async ML service is not reachable via RabbitMQ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, _, _, _, _, _, jobWorker, mlClient := setupPredictionControllerWithMocks()
			tt.setupMock(jobWorker, mlClient)

			router := setupPredictionTestRouter()
			router.GET("/prediction/health", controller.TestMLConnection)

			req := httptest.NewRequest("GET", "/prediction/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			jobWorker.AssertExpectations(t)
			mlClient.AssertExpectations(t)
		})
	}
}

func TestGetJobStatus(t *testing.T) {
	tests := []struct {
		name           string
		jobID          string
		userID         uint
		setupMock      func(*mocks.MockPredictionJobRepository, *mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful status retrieval",
			jobID:  "test-job-id",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, predRepo *mocks.MockPredictionRepository) {
				job := &models.PredictionJob{
					ID:     "test-job-id",
					UserID: 1,
					Status: models.JobStatusCompleted,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Job status retrieved successfully",
		},
		{
			name:   "job not found",
			jobID:  "nonexistent-job",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, predRepo *mocks.MockPredictionRepository) {
				jobRepo.On("GetJobByID", "nonexistent-job").Return(nil, errors.New("job not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Job not found",
		},
		{
			name:   "job belongs to different user",
			jobID:  "test-job-id",
			userID: 2,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, predRepo *mocks.MockPredictionRepository) {
				job := &models.PredictionJob{
					ID:     "test-job-id",
					UserID: 1,
					Status: models.JobStatusPending,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "Job belongs to a different user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, jobRepo, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(jobRepo, predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/job/:job_id/status", controller.GetJobStatus)

			req := httptest.NewRequest("GET", "/prediction/job/"+tt.jobID+"/status", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			jobRepo.AssertExpectations(t)
			predRepo.AssertExpectations(t)
		})
	}
}

func TestGetUserPredictions(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		limit          string
		setupMock      func(*mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful retrieval",
			userID: 1,
			limit:  "5",
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				predictions := []models.Prediction{
					{ID: 1, UserID: 1, RiskScore: 0.15},
					{ID: 2, UserID: 1, RiskScore: 0.20},
				}
				predRepo.On("GetPredictionsByUserID", uint(1), 5).Return(predictions, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction history retrieved successfully",
		},
		{
			name:   "default limit when empty",
			userID: 1,
			limit:  "",
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				predictions := []models.Prediction{}
				predRepo.On("GetPredictionsByUserID", uint(1), 10).Return(predictions, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction history retrieved successfully",
		},
		{
			name:           "invalid limit",
			userID:         1,
			limit:          "invalid",
			setupMock:      func(predRepo *mocks.MockPredictionRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid limit parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/me", controller.GetUserPredictions)

			url := "/prediction/me"
			if tt.limit != "" {
				url += "?limit=" + tt.limit
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
		})
	}
}

func TestGetPredictionByID(t *testing.T) {
	tests := []struct {
		name           string
		predictionID   string
		userID         uint
		setupMock      func(*mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:         "successful retrieval",
			predictionID: "1",
			userID:       1,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				prediction := &models.Prediction{ID: 1, UserID: 1, RiskScore: 0.15}
				predRepo.On("GetPredictionByID", uint(1)).Return(prediction, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction retrieved successfully",
		},
		{
			name:           "invalid prediction ID",
			predictionID:   "invalid",
			userID:         1,
			setupMock:      func(predRepo *mocks.MockPredictionRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid prediction ID",
		},
		{
			name:         "prediction not found",
			predictionID: "999",
			userID:       1,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				predRepo.On("GetPredictionByID", uint(999)).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Prediction not found",
		},
		{
			name:         "forbidden access",
			predictionID: "1",
			userID:       2,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				prediction := &models.Prediction{ID: 1, UserID: 1, RiskScore: 0.15}
				predRepo.On("GetPredictionByID", uint(1)).Return(prediction, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "Access denied: prediction belongs to a different user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/:id", controller.GetPredictionByID)

			req := httptest.NewRequest("GET", "/prediction/"+tt.predictionID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
		})
	}
}

func TestDeletePrediction(t *testing.T) {
	tests := []struct {
		name           string
		predictionID   string
		userID         uint
		setupMock      func(*mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:         "successful deletion",
			predictionID: "1",
			userID:       1,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				prediction := &models.Prediction{ID: 1, UserID: 1, RiskScore: 0.15}
				predRepo.On("GetPredictionByID", uint(1)).Return(prediction, nil)
				predRepo.On("DeletePrediction", uint(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction deleted successfully",
		},
		{
			name:           "invalid prediction ID",
			predictionID:   "invalid",
			userID:         1,
			setupMock:      func(predRepo *mocks.MockPredictionRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid prediction ID",
		},
		{
			name:         "prediction not found",
			predictionID: "999",
			userID:       1,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				predRepo.On("GetPredictionByID", uint(999)).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Prediction not found",
		},
		{
			name:         "forbidden deletion",
			predictionID: "1",
			userID:       2,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				prediction := &models.Prediction{ID: 1, UserID: 1, RiskScore: 0.15}
				predRepo.On("GetPredictionByID", uint(1)).Return(prediction, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "Access denied: prediction belongs to a different user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.DELETE("/prediction/:id", controller.DeletePrediction)

			req := httptest.NewRequest("DELETE", "/prediction/"+tt.predictionID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
		})
	}
}

func TestGetPredictionsByDateRange(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		startDate      string
		endDate        string
		setupMock      func(*mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:      "successful retrieval",
			userID:    1,
			startDate: "2024-01-01",
			endDate:   "2024-01-31",
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				predictions := []models.Prediction{
					{ID: 1, UserID: 1, RiskScore: 0.15},
					{ID: 2, UserID: 1, RiskScore: 0.20},
				}
				startTime, _ := time.Parse("2006-01-02", "2024-01-01")
				endTime, _ := time.Parse("2006-01-02", "2024-01-31")
				endTime = endTime.Add(24 * time.Hour).Add(-time.Second)
				predRepo.On("GetPredictionsByUserIDAndDateRange", uint(1), startTime, endTime).Return(predictions, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction history retrieved successfully",
		},
		{
			name:           "invalid start date",
			userID:         1,
			startDate:      "invalid-date",
			endDate:        "2024-01-31",
			setupMock:      func(predRepo *mocks.MockPredictionRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid start date format",
		},
		{
			name:           "invalid end date",
			userID:         1,
			startDate:      "2024-01-01",
			endDate:        "invalid-date",
			setupMock:      func(predRepo *mocks.MockPredictionRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid end date format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/me/date-range", controller.GetPredictionsByDateRange)

			url := "/prediction/me/date-range?start_date=" + tt.startDate + "&end_date=" + tt.endDate
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
		})
	}
}

func TestGetPredictionScoreByDate(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		startDate      string
		endDate        string
		setupMock      func(*mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:      "successful retrieval",
			userID:    1,
			startDate: "2024-01-01",
			endDate:   "2024-01-31",
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				scores := []repository.PredictionScore{
					{RiskScore: 0.15, CreatedAt: time.Now()},
					{RiskScore: 0.18, CreatedAt: time.Now().AddDate(0, 0, -1)},
				}
				startTime, _ := time.Parse("2006-01-02", "2024-01-01")
				endTime, _ := time.Parse("2006-01-02", "2024-01-31")
				endTime = endTime.Add(24 * time.Hour).Add(-time.Second)
				predRepo.On("GetPredictionScoreByUserIDAndDateRange", uint(1), startTime, endTime).Return(scores, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction score retrieved successfully",
		},
		{
			name:           "invalid start date",
			userID:         1,
			startDate:      "invalid-date",
			endDate:        "2024-01-31",
			setupMock:      func(predRepo *mocks.MockPredictionRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid start date format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/me/score", controller.GetPredictionScoreByDate)

			url := "/prediction/me/score?start_date=" + tt.startDate + "&end_date=" + tt.endDate
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
		})
	}
}

func TestGetLatestPredictionExplanation(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMock      func(*mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful retrieval with existing explanations",
			userID: 1,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				prediction := &models.Prediction{
					ID:                                   1,
					UserID:                               1,
					RiskScore:                            0.15,
					AgeExplanation:                       "Age factor explanation",
					BMIExplanation:                       "BMI factor explanation",
					BrinkmanScoreExplanation:             "Brinkman score explanation",
					IsHypertensionExplanation:            "Hypertension explanation",
					IsCholesterolExplanation:             "Cholesterol explanation",
					IsBloodlineExplanation:               "Bloodline explanation",
					IsMacrosomicBabyExplanation:          "Macrosomic baby explanation",
					SmokingStatusExplanation:             "Smoking status explanation",
					PhysicalActivityFrequencyExplanation: "Physical activity explanation",
				}
				predRepo.On("GetLatestPredictionByUserID", uint(1)).Return(prediction, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Latest prediction explanation retrieved successfully",
		},
		{
			name:   "no prediction found",
			userID: 1,
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				predRepo.On("GetLatestPredictionByUserID", uint(1)).Return(nil, errors.New("no prediction found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "No prediction found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, _, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/me/latest", controller.GetLatestPredictionExplanation)

			req := httptest.NewRequest("GET", "/prediction/me/latest", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			predRepo.AssertExpectations(t)
		})
	}
}

func TestGetUserJobs(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		status         string
		limit          string
		setupMock      func(*mocks.MockPredictionJobRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful retrieval with status filter",
			userID: 1,
			status: "completed",
			limit:  "5",
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository) {
				jobs := []*models.PredictionJob{
					{ID: "job1", UserID: 1, Status: "completed"},
					{ID: "job2", UserID: 1, Status: "completed"},
				}
				jobRepo.On("GetJobsByUserIDAndStatus", uint(1), "completed", 5).Return(jobs, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Jobs retrieved successfully",
		},
		{
			name:   "successful retrieval without status filter",
			userID: 1,
			status: "",
			limit:  "10",
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository) {
				jobs := []*models.PredictionJob{
					{ID: "job1", UserID: 1, Status: "completed"},
					{ID: "job2", UserID: 1, Status: "pending"},
				}
				jobRepo.On("GetJobsByUserID", uint(1), 10).Return(jobs, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Jobs retrieved successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, _, _, _, _, jobRepo, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(jobRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/jobs", controller.GetUserJobs)

			url := "/prediction/jobs?"
			if tt.status != "" {
				url += "status=" + tt.status + "&"
			}
			if tt.limit != "" {
				url += "limit=" + tt.limit
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			jobRepo.AssertExpectations(t)
		})
	}
}

func TestCancelJob(t *testing.T) {
	tests := []struct {
		name           string
		jobID          string
		userID         uint
		setupMock      func(*mocks.MockPredictionJobRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful job cancellation",
			jobID:  "test-job-id",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository) {
				job := &models.PredictionJob{
					ID:     "test-job-id",
					UserID: 1,
					Status: models.JobStatusPending,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)
				jobRepo.On("CancelJob", "test-job-id").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Job cancelled successfully",
		},
		{
			name:   "job not found",
			jobID:  "nonexistent-job",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository) {
				jobRepo.On("GetJobByID", "nonexistent-job").Return(nil, errors.New("job not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Job not found",
		},
		{
			name:   "cannot cancel submitted job",
			jobID:  "test-job-id",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository) {
				job := &models.PredictionJob{
					ID:     "test-job-id",
					UserID: 1,
					Status: models.JobStatusSubmitted,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Cannot cancel job that has been submitted to ML service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, _, _, _, _, jobRepo, _, _ := setupPredictionControllerWithMocks()
			tt.setupMock(jobRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.POST("/prediction/job/:job_id/cancel", controller.CancelJob)

			req := httptest.NewRequest("POST", "/prediction/job/"+tt.jobID+"/cancel", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			jobRepo.AssertExpectations(t)
		})
	}
}

func TestGetJobResult(t *testing.T) {
	tests := []struct {
		name           string
		jobID          string
		userID         uint
		setupMock      func(*mocks.MockPredictionJobRepository, *mocks.MockPredictionJobWorker, *mocks.MockPredictionRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful what-if result retrieval",
			jobID:  "test-job-id",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, predRepo *mocks.MockPredictionRepository) {
				job := &models.PredictionJob{
					ID:       "test-job-id",
					UserID:   1,
					Status:   models.JobStatusCompleted,
					IsWhatIf: true,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)

				result := map[string]interface{}{
					"risk_score":      0.15,
					"risk_percentage": 15.0,
					"timestamp":       time.Now(),
				}
				jobWorker.On("GetWhatIfResult", "test-job-id").Return(result, true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "What-if prediction result retrieved successfully",
		},
		{
			name:   "successful regular prediction result retrieval",
			jobID:  "test-job-id",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, predRepo *mocks.MockPredictionRepository) {
				predictionID := uint(1)
				job := &models.PredictionJob{
					ID:           "test-job-id",
					UserID:       1,
					Status:       models.JobStatusCompleted,
					IsWhatIf:     false,
					PredictionID: &predictionID,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)

				prediction := &models.Prediction{
					ID:        1,
					UserID:    1,
					RiskScore: 0.15,
				}
				predRepo.On("GetPredictionByID", uint(1)).Return(prediction, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Job result retrieved successfully",
		},
		{
			name:   "job not found",
			jobID:  "nonexistent-job",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, predRepo *mocks.MockPredictionRepository) {
				jobRepo.On("GetJobByID", "nonexistent-job").Return(nil, errors.New("job not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Job not found",
		},
		{
			name:   "what-if result not found in cache",
			jobID:  "test-job-id",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, predRepo *mocks.MockPredictionRepository) {
				job := &models.PredictionJob{
					ID:       "test-job-id",
					UserID:   1,
					Status:   models.JobStatusCompleted,
					IsWhatIf: true,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)
				jobWorker.On("GetWhatIfResult", "test-job-id").Return(map[string]interface{}{}, false, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "What-if result has expired or not found",
		},
		{
			name:   "job not completed yet",
			jobID:  "test-job-id",
			userID: 1,
			setupMock: func(jobRepo *mocks.MockPredictionJobRepository, jobWorker *mocks.MockPredictionJobWorker, predRepo *mocks.MockPredictionRepository) {
				job := &models.PredictionJob{
					ID:     "test-job-id",
					UserID: 1,
					Status: models.JobStatusPending,
				}
				jobRepo.On("GetJobByID", "test-job-id").Return(job, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Job is not completed yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, jobRepo, jobWorker, _ := setupPredictionControllerWithMocks()
			tt.setupMock(jobRepo, jobWorker, predRepo)

			router := setupPredictionTestRouter()
			router.Use(addPredictionAuthMiddleware(tt.userID))
			router.GET("/prediction/job/:job_id/result", controller.GetJobResult)

			req := httptest.NewRequest("GET", "/prediction/job/"+tt.jobID+"/result", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			jobRepo.AssertExpectations(t)
			jobWorker.AssertExpectations(t)
			predRepo.AssertExpectations(t)
		})
	}
}
