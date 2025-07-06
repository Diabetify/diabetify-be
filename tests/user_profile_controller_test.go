package tests

import (
	"bytes"
	"diabetify/internal/controllers"
	"diabetify/internal/models"
	"diabetify/tests/mocks"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test helper functions
func setupProfileTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupProfileControllerWithMock() (*controllers.UserProfileController, *mocks.MockUserProfileRepository) {
	mockRepo := new(mocks.MockUserProfileRepository)
	controller := controllers.NewUserProfileController(mockRepo)
	return controller, mockRepo
}

func addProfileAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestNewUserProfileController(t *testing.T) {
	mockRepo := new(mocks.MockUserProfileRepository)
	controller := controllers.NewUserProfileController(mockRepo)

	assert.NotNil(t, controller)
}

func TestGetUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMock      func(*mocks.MockUserProfileRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful retrieval",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileRepository) {
				weight := 70
				height := 170
				bmi := 24.2
				profile := &models.UserProfile{
					ID:     1,
					UserID: 1,
					Weight: &weight,
					Height: &height,
					BMI:    &bmi,
				}
				m.On("FindByUserID", uint(1)).Return(profile, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "User profile retrieved successfully",
		},
		{
			name:   "profile not found",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("FindByUserID", uint(1)).Return(nil, errors.New("profile not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Profile not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupProfileControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.GET("/profile", controller.GetUserProfile)

			req := httptest.NewRequest("GET", "/profile", nil)
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

func TestGetUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.GET("/profile", controller.GetUserProfile)

	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}

func TestCreateUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockUserProfileRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful creation with BMI calculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 70.0,
				"height": 170.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("Create", mock.MatchedBy(func(profile *models.UserProfile) bool {
					return profile.UserID == 1 &&
						profile.Weight != nil && *profile.Weight == 70.0 &&
						profile.Height != nil && *profile.Height == 170.0 &&
						profile.BMI != nil && *profile.BMI == 24.2
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedMsg:    "Profile created successfully",
		},
		{
			name:   "successful creation without BMI calculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 70.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("Create", mock.MatchedBy(func(profile *models.UserProfile) bool {
					return profile.UserID == 1 &&
						profile.Weight != nil && *profile.Weight == 70.0 &&
						profile.BMI == nil
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedMsg:    "Profile created successfully",
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockUserProfileRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name:   "repository error",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 70.0,
				"height": 170.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("Create", mock.AnythingOfType("*models.UserProfile")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to create profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupProfileControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.POST("/profile", controller.CreateUserProfile)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("POST", "/profile", bytes.NewBuffer(body))
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

func TestCreateUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.POST("/profile", controller.CreateUserProfile)

	requestBody := map[string]interface{}{
		"weight": 70.0,
		"height": 170.0,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}

func TestUpdateUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockUserProfileRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful update with BMI recalculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 75.0,
				"height": 175.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				existingProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
				}
				m.On("FindByUserID", uint(1)).Return(existingProfile, nil)
				m.On("Update", mock.MatchedBy(func(profile *models.UserProfile) bool {
					return profile.ID == 1 &&
						profile.UserID == 1 &&
						profile.Weight != nil && *profile.Weight == 75.0 &&
						profile.Height != nil && *profile.Height == 175.0 &&
						profile.BMI != nil && *profile.BMI == 24.5
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile updated successfully",
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockUserProfileRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name:   "profile not found",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 75.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("FindByUserID", uint(1)).Return(nil, errors.New("profile not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Profile not found",
		},
		{
			name:   "repository update error",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 75.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				existingProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
				}
				m.On("FindByUserID", uint(1)).Return(existingProfile, nil)
				m.On("Update", mock.AnythingOfType("*models.UserProfile")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to update profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupProfileControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.PUT("/profile", controller.UpdateUserProfile)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(body))
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

func TestDeleteUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMock      func(*mocks.MockUserProfileRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful deletion",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("DeleteByUserID", uint(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile deleted successfully",
		},
		{
			name:   "repository error",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("DeleteByUserID", uint(1)).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to delete profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupProfileControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.DELETE("/profile", controller.DeleteUserProfile)

			req := httptest.NewRequest("DELETE", "/profile", nil)
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

func TestDeleteUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.DELETE("/profile", controller.DeleteUserProfile)

	req := httptest.NewRequest("DELETE", "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}

func TestPatchUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockUserProfileRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful patch with BMI recalculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 80.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				height := 170
				existingProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
					Height: &height,
				}
				m.On("FindByUserID", uint(1)).Return(existingProfile, nil).Once()
				m.On("Patch", uint(1), mock.MatchedBy(func(data map[string]interface{}) bool {
					return data["bmi"] != nil
				})).Return(nil)

				// Mock the second FindByUserID call for returning updated profile
				updatedHeight := 170
				updatedWeight := 80
				updatedBMI := 27.7
				updatedProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
					Height: &updatedHeight,
					Weight: &updatedWeight,
					BMI:    &updatedBMI,
				}
				m.On("FindByUserID", uint(1)).Return(updatedProfile, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile patched successfully",
		},
		{
			name:   "successful patch without BMI recalculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"bloodline": true,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				existingProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
				}
				m.On("FindByUserID", uint(1)).Return(existingProfile, nil).Once()
				m.On("Patch", uint(1), map[string]interface{}{
					"bloodline": true,
				}).Return(nil)

				updatedProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
				}
				m.On("FindByUserID", uint(1)).Return(updatedProfile, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile patched successfully",
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockUserProfileRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name:   "profile not found",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 80.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				m.On("FindByUserID", uint(1)).Return(nil, errors.New("profile not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Profile not found",
		},
		{
			name:   "patch repository error",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 80.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				existingProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
				}
				m.On("FindByUserID", uint(1)).Return(existingProfile, nil)
				m.On("Patch", uint(1), mock.AnythingOfType("map[string]interface {}")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to update profile",
		},
		{
			name:   "failed to retrieve updated profile",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 80.0,
			},
			setupMock: func(m *mocks.MockUserProfileRepository) {
				existingProfile := &models.UserProfile{
					ID:     1,
					UserID: 1,
				}
				m.On("FindByUserID", uint(1)).Return(existingProfile, nil).Once()
				m.On("Patch", uint(1), mock.AnythingOfType("map[string]interface {}")).Return(nil)
				m.On("FindByUserID", uint(1)).Return(nil, errors.New("failed to retrieve")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to retrieve updated profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo := setupProfileControllerWithMock()
			tt.setupMock(mockRepo)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.PATCH("/profile", controller.PatchUserProfile)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("PATCH", "/profile", bytes.NewBuffer(body))
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

func TestPatchUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.PATCH("/profile", controller.PatchUserProfile)

	requestBody := map[string]interface{}{
		"weight": 80.0,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("PATCH", "/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}
