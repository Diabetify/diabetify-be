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

// Test helper functions
func setupPredictionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupPredictionControllerWithMocks() (*controllers.PredictionController, *mocks.MockPredictionRepository, *mocks.MockUserRepository, *mocks.MockUserProfileRepository, *mocks.MockActivityRepository, *mocks.MockMLClient) {
	mockPredRepo := new(mocks.MockPredictionRepository)
	mockUserRepo := new(mocks.MockUserRepository)
	mockProfileRepo := new(mocks.MockUserProfileRepository)
	mockActivityRepo := new(mocks.MockActivityRepository)
	mockMLClient := new(mocks.MockMLClient)

	controller := controllers.NewPredictionController(
		mockPredRepo,
		mockUserRepo,
		mockProfileRepo,
		mockActivityRepo,
		mockMLClient,
	)

	return controller, mockPredRepo, mockUserRepo, mockProfileRepo, mockActivityRepo, mockMLClient
}

func addPredictionAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestNewPredictionController(t *testing.T) {
	controller, _, _, _, _, _ := setupPredictionControllerWithMocks()
	assert.NotNil(t, controller)
}

func TestMakePrediction(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMocks     func(*mocks.MockPredictionRepository, *mocks.MockUserRepository, *mocks.MockUserProfileRepository, *mocks.MockActivityRepository, *mocks.MockMLClient)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful prediction",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, mlClient *mocks.MockMLClient) {
				// Mock user data
				dob := "1990-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				// Mock profile data
				bmi := 25.0
				hypertension := false
				cholesterol := false
				macrosomicBaby := 0
				bloodline := false
				height := 170
				profile := &models.UserProfile{
					BMI:            &bmi,
					Hypertension:   &hypertension,
					Cholesterol:    &cholesterol,
					MacrosomicBaby: &macrosomicBaby,
					Bloodline:      &bloodline,
					Height:         &height,
					CreatedAt:      time.Now().AddDate(0, 0, -30),
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)

				// Mock activity data
				activityRepo.On("GetActivitiesByUserIDAndTypeAndDateRange", uint(1), "smoke", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Activity{}, nil)
				activityRepo.On("GetActivitiesByUserIDAndTypeAndDateRange", uint(1), "workout", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Activity{}, nil)

				// Mock ML response
				mlResponse := &models.PredictionResponse{
					Prediction:  0.15,
					Explanation: map[string]models.ExplanationItem{},
					ElapsedTime: 50,
					Timestamp:   time.Now(),
				}
				mlClient.On("Predict", mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("[]float64")).Return(mlResponse, nil)

				// Mock save prediction
				predRepo.On("SavePrediction", mock.AnythingOfType("*models.Prediction")).Return(nil)

				// Mock update last prediction time
				userRepo.On("UpdateLastPredictionTime", uint(1), mock.AnythingOfType("*time.Time")).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction successful via gRPC using your profile data",
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, mlClient *mocks.MockMLClient) {
				userRepo.On("GetUserByID", uint(999)).Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name:   "user profile not found",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, mlClient *mocks.MockMLClient) {
				dob := "1990-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)
				profileRepo.On("FindByUserID", uint(1)).Return(nil, errors.New("profile not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User profile not found. Please complete your profile first.",
		},
		{
			name:   "incomplete profile data",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, mlClient *mocks.MockMLClient) {
				dob := "1990-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				// Profile with missing BMI
				profile := &models.UserProfile{
					BMI: nil, // Missing BMI
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Incomplete profile data for prediction",
		},
		{
			name:   "ML service error",
			userID: 1,
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, mlClient *mocks.MockMLClient) {
				dob := "1990-01-01"
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
				height := 170
				profile := &models.UserProfile{
					BMI:            &bmi,
					Hypertension:   &hypertension,
					Cholesterol:    &cholesterol,
					MacrosomicBaby: &macrosomicBaby,
					Bloodline:      &bloodline,
					Height:         &height,
					CreatedAt:      time.Now().AddDate(0, 0, -30),
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)

				activityRepo.On("GetActivitiesByUserIDAndTypeAndDateRange", uint(1), "smoke", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Activity{}, nil)
				activityRepo.On("GetActivitiesByUserIDAndTypeAndDateRange", uint(1), "workout", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Activity{}, nil)

				mlClient.On("Predict", mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("[]float64")).Return(nil, errors.New("ML service unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Prediction failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, userRepo, profileRepo, activityRepo, mlClient := setupPredictionControllerWithMocks()
			tt.setupMocks(predRepo, userRepo, profileRepo, activityRepo, mlClient)

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
			mlClient.AssertExpectations(t)
		})
	}
}

func TestMakePredictionUnauthorized(t *testing.T) {
	controller, _, _, _, _, _ := setupPredictionControllerWithMocks()
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
		setupMocks     func(*mocks.MockPredictionRepository, *mocks.MockUserRepository, *mocks.MockUserProfileRepository, *mocks.MockActivityRepository, *mocks.MockMLClient)
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
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, mlClient *mocks.MockMLClient) {
				dob := "1990-01-01"
				user := &models.User{
					ID:  1,
					DOB: &dob,
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)

				macrosomicBaby := 0
				bloodline := false
				height := 170
				yearOfSmoking := 5
				profile := &models.UserProfile{
					MacrosomicBaby: &macrosomicBaby,
					Bloodline:      &bloodline,
					Height:         &height,
					YearOfSmoking:  &yearOfSmoking,
				}
				profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)

				mlResponse := &models.PredictionResponse{
					Prediction:  0.25,
					Explanation: map[string]models.ExplanationItem{},
					ElapsedTime: 45,
					Timestamp:   time.Now(),
				}
				mlClient.On("Predict", mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("[]float64")).Return(mlResponse, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Prediction successful via gRPC using your profile data",
		},
		{
			name:   "invalid input format",
			userID: 1,
			requestBody: map[string]interface{}{
				"smoking_status": "invalid", // Should be int
			},
			setupMocks: func(predRepo *mocks.MockPredictionRepository, userRepo *mocks.MockUserRepository, profileRepo *mocks.MockUserProfileRepository, activityRepo *mocks.MockActivityRepository, mlClient *mocks.MockMLClient) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid input format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, userRepo, profileRepo, activityRepo, mlClient := setupPredictionControllerWithMocks()
			tt.setupMocks(predRepo, userRepo, profileRepo, activityRepo, mlClient)

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
			mlClient.AssertExpectations(t)
		})
	}
}

func TestTestMLConnection(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockMLClient)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "ML service healthy",
			setupMock: func(mlClient *mocks.MockMLClient) {
				mlClient.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "ML service is healthy via gRPC",
		},
		{
			name: "ML service unhealthy",
			setupMock: func(mlClient *mocks.MockMLClient) {
				mlClient.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(errors.New("connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "ML service is not reachable via gRPC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, _, _, _, _, mlClient := setupPredictionControllerWithMocks()
			tt.setupMock(mlClient)

			router := setupPredictionTestRouter()
			router.GET("/prediction/predict/health", controller.TestMLConnection)

			req := httptest.NewRequest("GET", "/prediction/predict/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mlClient.AssertExpectations(t)
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
		{
			name:   "repository error",
			userID: 1,
			limit:  "5",
			setupMock: func(predRepo *mocks.MockPredictionRepository) {
				predRepo.On("GetPredictionsByUserID", uint(1), 5).Return([]models.Prediction{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to retrieve prediction history",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, predRepo, _, _, _, _ := setupPredictionControllerWithMocks()
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
			controller, predRepo, _, _, _, _ := setupPredictionControllerWithMocks()
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
			controller, predRepo, _, _, _, _ := setupPredictionControllerWithMocks()
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
			controller, predRepo, _, _, _, _ := setupPredictionControllerWithMocks()
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
			controller, predRepo, _, _, _, _ := setupPredictionControllerWithMocks()
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
			controller, predRepo, _, _, _, _ := setupPredictionControllerWithMocks()
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

// Benchmark tests
func BenchmarkMakePrediction(b *testing.B) {
	controller, predRepo, userRepo, profileRepo, activityRepo, mlClient := setupPredictionControllerWithMocks()

	// Setup mocks for benchmark
	dob := "1990-01-01"
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
	height := 170
	profile := &models.UserProfile{
		BMI:            &bmi,
		Hypertension:   &hypertension,
		Cholesterol:    &cholesterol,
		MacrosomicBaby: &macrosomicBaby,
		Bloodline:      &bloodline,
		Height:         &height,
		CreatedAt:      time.Now().AddDate(0, 0, -30),
	}
	profileRepo.On("FindByUserID", uint(1)).Return(profile, nil)

	activityRepo.On("GetActivitiesByUserIDAndTypeAndDateRange", uint(1), "smoke", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Activity{}, nil)
	activityRepo.On("GetActivitiesByUserIDAndTypeAndDateRange", uint(1), "workout", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.Activity{}, nil)

	mlResponse := &models.PredictionResponse{
		Prediction:  0.15,
		Explanation: map[string]models.ExplanationItem{},
		ElapsedTime: 50,
		Timestamp:   time.Now(),
	}
	mlClient.On("Predict", mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("[]float64")).Return(mlResponse, nil)

	predRepo.On("SavePrediction", mock.AnythingOfType("*models.Prediction")).Return(nil)
	userRepo.On("UpdateLastPredictionTime", uint(1), mock.AnythingOfType("*time.Time")).Return(nil)

	router := setupPredictionTestRouter()
	router.Use(addPredictionAuthMiddleware(1))
	router.POST("/prediction", controller.MakePrediction)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/prediction", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
