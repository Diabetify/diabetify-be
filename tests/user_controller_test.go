package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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
func setupUserTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupUserControllerWithMocks() (*controllers.UserController, *mocks.MockUserRepository, *mocks.MockResetPasswordRepository) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockResetPasswordRepo := new(mocks.MockResetPasswordRepository)
	controller := controllers.NewUserController(mockUserRepo, mockResetPasswordRepo)
	return controller, mockUserRepo, mockResetPasswordRepo
}

func addUserAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// Helper function to create a test password hash
func createTestPasswordHash(password string) string {
	// Use a fixed salt for testing
	salt := make([]byte, 8)
	// Fill with known values for consistency
	for i := range salt {
		salt[i] = byte(i)
	}

	// SHA256
	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return hex.EncodeToString(salt) + hex.EncodeToString(hash)
}

func TestLoginUser(t *testing.T) {
	os.Setenv("JWT_SECRET_KEY", "test-secret-key")
	defer os.Unsetenv("JWT_SECRET_KEY")

	// Create a proper test password hash
	testPassword := "password123"
	testPasswordHash := createTestPasswordHash(testPassword)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockUserRepository, *mocks.MockResetPasswordRepository)
		expectedStatus int
		expectedMsg    string
		checkToken     bool
	}{
		{
			name: "successful login",
			requestBody: map[string]interface{}{
				"email":    "john@example.com",
				"password": testPassword,
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				user := &models.User{
					ID:       1,
					Email:    "john@example.com",
					Password: testPasswordHash,
				}
				userRepo.On("GetUserByEmail", "john@example.com").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "User logged in successfully",
			checkToken:     true,
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "password123",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				userRepo.On("GetUserByEmail", "nonexistent@example.com").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
			checkToken:     false,
		},
		{
			name: "incorrect password",
			requestBody: map[string]interface{}{
				"email":    "john@example.com",
				"password": "wrongpassword",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				user := &models.User{
					ID:       1,
					Email:    "john@example.com",
					Password: testPasswordHash,
				}
				userRepo.On("GetUserByEmail", "john@example.com").Return(user, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Unauthorized",
			checkToken:     false,
		},
		{
			name: "invalid request data",
			requestBody: map[string]interface{}{
				"email": "john@example.com",
				// Missing password
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
			checkToken:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, userRepo, resetRepo := setupUserControllerWithMocks()
			tt.setupMocks(userRepo, resetRepo)

			router := setupUserTestRouter()
			router.POST("/users/login", controller.LoginUser)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/users/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			if tt.checkToken {
				assert.NotNil(t, response["data"])
				assert.IsType(t, "", response["data"])
			}

			userRepo.AssertExpectations(t)
			resetRepo.AssertExpectations(t)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMocks     func(*mocks.MockUserRepository, *mocks.MockResetPasswordRepository)
		hasAuth        bool
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful get current user",
			userID: 1,
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				user := &models.User{
					ID:    1,
					Name:  "John Doe",
					Email: "john@example.com",
				}
				userRepo.On("GetUserByID", uint(1)).Return(user, nil)
			},
			hasAuth:        true,
			expectedStatus: http.StatusOK,
			expectedMsg:    "User information retrieved successfully",
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				userRepo.On("GetUserByID", uint(999)).Return(nil, errors.New("user not found"))
			},
			hasAuth:        true,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name:           "unauthorized - no user_id in context",
			userID:         0,
			setupMocks:     func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {},
			hasAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, userRepo, resetRepo := setupUserControllerWithMocks()
			tt.setupMocks(userRepo, resetRepo)

			router := setupUserTestRouter()
			if tt.hasAuth {
				router.Use(addUserAuthMiddleware(tt.userID))
			}
			router.GET("/users/me", controller.GetCurrentUser)

			req := httptest.NewRequest("GET", "/users/me", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			userRepo.AssertExpectations(t)
			resetRepo.AssertExpectations(t)
		})
	}
}

func TestForgotPassword(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockUserRepository, *mocks.MockResetPasswordRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful forgot password",
			requestBody: map[string]interface{}{
				"email": "user@example.com",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				user := &models.User{
					ID:    1,
					Email: "user@example.com",
				}
				userRepo.On("GetUserByEmail", "user@example.com").Return(user, nil)
				resetRepo.On("DeleteByEmail", "user@example.com").Return(nil)
				resetRepo.On("CreateResetPassword", mock.AnythingOfType("*models.ResetPassword")).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Code sent successfully",
		},
		{
			name: "email does not exist",
			requestBody: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				userRepo.On("GetUserByEmail", "nonexistent@example.com").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Email's does not exist",
		},
		{
			name:        "invalid request data",
			requestBody: map[string]interface{}{
				// Missing email
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name: "database error when creating reset password",
			requestBody: map[string]interface{}{
				"email": "user@example.com",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				user := &models.User{
					ID:    1,
					Email: "user@example.com",
				}
				userRepo.On("GetUserByEmail", "user@example.com").Return(user, nil)
				resetRepo.On("DeleteByEmail", "user@example.com").Return(nil)
				resetRepo.On("CreateResetPassword", mock.AnythingOfType("*models.ResetPassword")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to create forget password code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, userRepo, resetRepo := setupUserControllerWithMocks()
			tt.setupMocks(userRepo, resetRepo)

			router := setupUserTestRouter()
			router.POST("/users/forgot-password", controller.ForgotPassword)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/users/forgot-password", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			userRepo.AssertExpectations(t)
			resetRepo.AssertExpectations(t)
		})
	}
}

func TestResetPassword(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockUserRepository, *mocks.MockResetPasswordRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful password reset",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				resetRecord := &models.ResetPassword{
					Email:     "user@example.com",
					Code:      "123456",
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}
				resetRepo.On("FindByEmailAndCode", "user@example.com", "123456").Return(resetRecord, nil)

				user := &models.User{
					ID:    1,
					Email: "user@example.com",
				}
				userRepo.On("GetUserByEmail", "user@example.com").Return(user, nil)
				userRepo.On("UpdateUser", mock.AnythingOfType("*models.User")).Return(nil)
				resetRepo.On("DeleteByEmail", "user@example.com").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Password has been reset successfully",
		},
		{
			name: "invalid or expired code",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "wrong123",
				"new_password": "newpassword123",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				resetRepo.On("FindByEmailAndCode", "user@example.com", "wrong123").Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid or expired code",
		},
		{
			name: "expired code",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				resetRecord := &models.ResetPassword{
					Email:     "user@example.com",
					Code:      "123456",
					ExpiresAt: time.Now().Add(-10 * time.Minute), // Expired
				}
				resetRepo.On("FindByEmailAndCode", "user@example.com", "123456").Return(resetRecord, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Code has expired",
		},
		{
			name: "password too short",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "short",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				resetRecord := &models.ResetPassword{
					Email:     "user@example.com",
					Code:      "123456",
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}
				resetRepo.On("FindByEmailAndCode", "user@example.com", "123456").Return(resetRecord, nil)

				user := &models.User{
					ID:    1,
					Email: "user@example.com",
				}
				userRepo.On("GetUserByEmail", "user@example.com").Return(user, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Password must be at least 8 characters",
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				resetRecord := &models.ResetPassword{
					Email:     "user@example.com",
					Code:      "123456",
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}
				resetRepo.On("FindByEmailAndCode", "user@example.com", "123456").Return(resetRecord, nil)
				userRepo.On("GetUserByEmail", "user@example.com").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name: "database error when updating password",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				resetRecord := &models.ResetPassword{
					Email:     "user@example.com",
					Code:      "123456",
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}
				resetRepo.On("FindByEmailAndCode", "user@example.com", "123456").Return(resetRecord, nil)

				user := &models.User{
					ID:    1,
					Email: "user@example.com",
				}
				userRepo.On("GetUserByEmail", "user@example.com").Return(user, nil)
				userRepo.On("UpdateUser", mock.AnythingOfType("*models.User")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to update password",
		},
		{
			name: "invalid request data",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
				"code":  "123456",
				// Missing new_password
			},
			setupMocks: func(userRepo *mocks.MockUserRepository, resetRepo *mocks.MockResetPasswordRepository) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, userRepo, resetRepo := setupUserControllerWithMocks()
			tt.setupMocks(userRepo, resetRepo)

			router := setupUserTestRouter()
			router.POST("/users/reset-password", controller.ResetPassword)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/users/reset-password", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			userRepo.AssertExpectations(t)
			resetRepo.AssertExpectations(t)
		})
	}
}
