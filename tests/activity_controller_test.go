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
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test helper functions
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupControllerWithMock() (*controllers.ActivityController, *mocks.MockActivityRepository) {
	mockRepo := new(mocks.MockActivityRepository)
	controller := controllers.NewActivityController(mockRepo)
	return controller, mockRepo
}

func addAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestNewActivityController(t *testing.T) {
	mockRepo := new(mocks.MockActivityRepository)
	controller := controllers.NewActivityController(mockRepo)

	assert.NotNil(t, controller)
}

func TestCreateActivity(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockActivityRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful creation",
			userID: 1,
			requestBody: map[string]interface{}{
				"activity_type": "workout",
				"value":         30,
				"activity_date": "2024-01-01T10:00:00Z",
			},
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("Create", mock.AnythingOfType("*models.Activity")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedMsg:    "Activity created successfully",
		},
		{
			name:   "missing activity type",
			userID: 1,
			requestBody: map[string]interface{}{
				"value":         30,
				"activity_date": "2024-01-01T10:00:00Z",
			},
			setupMock:      func(m *mocks.MockActivityRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Activity type is required",
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockActivityRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name:   "repository error",
			userID: 1,
			requestBody: map[string]interface{}{
				"activity_type": "workout",
				"value":         30,
				"activity_date": "2024-01-01T10:00:00Z",
			},
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("Create", mock.AnythingOfType("*models.Activity")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to create activity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupTestRouter()
			router.Use(addAuthMiddleware(tt.userID))
			router.POST("/activity", controller.CreateActivity)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("POST", "/activity", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCreateActivityUnauthorized(t *testing.T) {
	controller, _ := setupControllerWithMock()
	router := setupTestRouter()
	router.POST("/activity", controller.CreateActivity)

	requestBody := map[string]interface{}{
		"activity_type": "workout",
		"value":         30,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/activity", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized access", response["message"])
}

func TestGetCurrentUserActivities(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		limit          string
		setupMock      func(*mocks.MockActivityRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful retrieval",
			userID: 1,
			limit:  "5",
			setupMock: func(m *mocks.MockActivityRepository) {
				activities := []models.Activity{
					{ID: 1, ActivityType: "smoke", UserID: 1, Value: 1},
					{ID: 2, ActivityType: "workout", UserID: 1, Value: 30},
				}
				m.On("FindAllByUserID", uint(1), 5).Return(activities, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Activities retrieved successfully",
		},
		{
			name:   "default limit when invalid",
			userID: 1,
			limit:  "invalid",
			setupMock: func(m *mocks.MockActivityRepository) {
				activities := []models.Activity{}
				m.On("FindAllByUserID", uint(1), 10).Return(activities, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Activities retrieved successfully",
		},
		{
			name:   "repository error",
			userID: 1,
			limit:  "5",
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("FindAllByUserID", uint(1), 5).Return([]models.Activity{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to retrieve activities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupTestRouter()
			router.Use(addAuthMiddleware(tt.userID))
			router.GET("/activity/me", controller.GetCurrentUserActivities)

			url := "/activity/me"
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
			assert.Equal(t, tt.expectedMsg, response["message"])

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetActivityByID(t *testing.T) {
	tests := []struct {
		name           string
		activityID     string
		userID         uint
		setupMock      func(*mocks.MockActivityRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:       "successful retrieval",
			activityID: "1",
			userID:     1,
			setupMock: func(m *mocks.MockActivityRepository) {
				activity := &models.Activity{ID: 1, ActivityType: "workout", UserID: 1, Value: 30}
				m.On("FindByID", uint(1)).Return(activity, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Activity retrieved successfully",
		},
		{
			name:           "invalid activity ID",
			activityID:     "invalid",
			userID:         1,
			setupMock:      func(m *mocks.MockActivityRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid activity ID",
		},
		{
			name:       "activity not found",
			activityID: "999",
			userID:     1,
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("FindByID", uint(999)).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Activity not found",
		},
		{
			name:       "forbidden access",
			activityID: "1",
			userID:     2,
			setupMock: func(m *mocks.MockActivityRepository) {
				activity := &models.Activity{ID: 1, ActivityType: "workout", UserID: 1, Value: 30}
				m.On("FindByID", uint(1)).Return(activity, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "You can only access your own activities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupTestRouter()
			router.Use(addAuthMiddleware(tt.userID))
			router.GET("/activity/:id", controller.GetActivityByID)

			req := httptest.NewRequest("GET", "/activity/"+tt.activityID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateActivity(t *testing.T) {
	tests := []struct {
		name           string
		activityID     string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockActivityRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:       "successful update",
			activityID: "1",
			userID:     1,
			requestBody: map[string]interface{}{
				"activity_type": "workout",
				"value":         45,
			},
			setupMock: func(m *mocks.MockActivityRepository) {
				existingActivity := &models.Activity{ID: 1, ActivityType: "workout", UserID: 1, Value: 30}
				m.On("FindByID", uint(1)).Return(existingActivity, nil)
				m.On("Update", mock.AnythingOfType("*models.Activity")).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Activity updated successfully",
		},
		{
			name:           "invalid activity ID",
			activityID:     "invalid",
			userID:         1,
			requestBody:    map[string]interface{}{},
			setupMock:      func(m *mocks.MockActivityRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid activity ID",
		},
		{
			name:       "activity not found",
			activityID: "999",
			userID:     1,
			requestBody: map[string]interface{}{
				"activity_type": "workout",
			},
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("FindByID", uint(999)).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Activity not found",
		},
		{
			name:       "forbidden update",
			activityID: "1",
			userID:     2,
			requestBody: map[string]interface{}{
				"activity_type": "workout",
			},
			setupMock: func(m *mocks.MockActivityRepository) {
				existingActivity := &models.Activity{ID: 1, ActivityType: "workout", UserID: 1, Value: 30}
				m.On("FindByID", uint(1)).Return(existingActivity, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "You can only update your own activities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupTestRouter()
			router.Use(addAuthMiddleware(tt.userID))
			router.PUT("/activity/:id", controller.UpdateActivity)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/activity/"+tt.activityID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteActivity(t *testing.T) {
	tests := []struct {
		name           string
		activityID     string
		userID         uint
		setupMock      func(*mocks.MockActivityRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:       "successful deletion",
			activityID: "1",
			userID:     1,
			setupMock: func(m *mocks.MockActivityRepository) {
				existingActivity := &models.Activity{ID: 1, ActivityType: "workout", UserID: 1, Value: 30}
				m.On("FindByID", uint(1)).Return(existingActivity, nil)
				m.On("Delete", uint(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Activity deleted successfully",
		},
		{
			name:           "invalid activity ID",
			activityID:     "invalid",
			userID:         1,
			setupMock:      func(m *mocks.MockActivityRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid activity ID",
		},
		{
			name:       "activity not found",
			activityID: "999",
			userID:     1,
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("FindByID", uint(999)).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Activity not found",
		},
		{
			name:       "forbidden deletion",
			activityID: "1",
			userID:     2,
			setupMock: func(m *mocks.MockActivityRepository) {
				existingActivity := &models.Activity{ID: 1, ActivityType: "workout", UserID: 1, Value: 30}
				m.On("FindByID", uint(1)).Return(existingActivity, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "You can only delete your own activities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupTestRouter()
			router.Use(addAuthMiddleware(tt.userID))
			router.DELETE("/activity/:id", controller.DeleteActivity)

			req := httptest.NewRequest("DELETE", "/activity/"+tt.activityID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetActivitiesByDateRange(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		startDate      string
		endDate        string
		setupMock      func(*mocks.MockActivityRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:      "successful retrieval",
			userID:    1,
			startDate: "2024-01-01",
			endDate:   "2024-01-31",
			setupMock: func(m *mocks.MockActivityRepository) {
				activities := []models.Activity{
					{ID: 1, ActivityType: "smoke", UserID: 1, Value: 1},
					{ID: 2, ActivityType: "workout", UserID: 1, Value: 30},
				}
				startTime, _ := time.Parse("2006-01-02", "2024-01-01")
				endTime, _ := time.Parse("2006-01-02", "2024-01-31")
				endTime = endTime.Add(24 * time.Hour)
				m.On("FindByUserIDAndActivityDateRange", uint(1), startTime, endTime).Return(activities, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Activities retrieved successfully",
		},
		{
			name:           "invalid start date",
			userID:         1,
			startDate:      "invalid-date",
			endDate:        "2024-01-31",
			setupMock:      func(m *mocks.MockActivityRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid start date format",
		},
		{
			name:           "invalid end date",
			userID:         1,
			startDate:      "2024-01-01",
			endDate:        "invalid-date",
			setupMock:      func(m *mocks.MockActivityRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid end date format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupTestRouter()
			router.Use(addAuthMiddleware(tt.userID))
			router.GET("/activity/me/date-range", controller.GetActivitiesByDateRange)

			url := "/activity/me/date-range?start_date=" + tt.startDate + "&end_date=" + tt.endDate
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCountUserActivities(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMock      func(*mocks.MockActivityRepository)
		expectedStatus int
		expectedMsg    string
		expectedCount  int64
	}{
		{
			name:   "successful count",
			userID: 1,
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("CountUserActivities", uint(1)).Return(int64(5), nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Activity count retrieved successfully",
			expectedCount:  5,
		},
		{
			name:   "repository error",
			userID: 1,
			setupMock: func(m *mocks.MockActivityRepository) {
				m.On("CountUserActivities", uint(1)).Return(int64(0), errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to count activities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupTestRouter()
			router.Use(addAuthMiddleware(tt.userID))
			router.GET("/activity/me/count", controller.CountUserActivities)

			req := httptest.NewRequest("GET", "/activity/me/count", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			if tt.expectedStatus == http.StatusOK {
				data := response["data"].(map[string]interface{})
				count := int64(data["count"].(float64))
				assert.Equal(t, tt.expectedCount, count)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
