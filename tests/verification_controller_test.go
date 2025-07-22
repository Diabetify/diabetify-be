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
func setupVerificationTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupVerificationControllerWithMocks() (*controllers.VerificationController, *mocks.MockVerificationRepository, *mocks.MockUserRepository) {
	mockVerificationRepo := new(mocks.MockVerificationRepository)
	mockUserRepo := new(mocks.MockUserRepository)
	controller := controllers.NewVerificationController(mockVerificationRepo, mockUserRepo)
	return controller, mockVerificationRepo, mockUserRepo
}

func TestNewVerificationController(t *testing.T) {
	controller, _, _ := setupVerificationControllerWithMocks()
	assert.NotNil(t, controller)
}

func TestSendVerificationCode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockVerificationRepository, *mocks.MockUserRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful verification code send",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				user := &models.User{
					ID:    1,
					Email: "test@example.com",
				}
				userRepo.On("GetUserByEmail", "test@example.com").Return(user, nil)
				verificationRepo.On("DeleteByEmail", "test@example.com").Return(nil)
				verificationRepo.On("CreateVerification", mock.AnythingOfType("*models.Verification")).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Verification code sent successfully",
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				userRepo.On("GetUserByEmail", "nonexistent@example.com").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name: "database error when creating verification",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				user := &models.User{
					ID:    1,
					Email: "test@example.com",
				}
				userRepo.On("GetUserByEmail", "test@example.com").Return(user, nil)
				verificationRepo.On("DeleteByEmail", "test@example.com").Return(nil)
				verificationRepo.On("CreateVerification", mock.AnythingOfType("*models.Verification")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to create verification code",
		},
		{
			name:        "missing email field",
			requestBody: map[string]interface{}{},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, verificationRepo, userRepo := setupVerificationControllerWithMocks()
			tt.setupMocks(verificationRepo, userRepo)

			router := setupVerificationTestRouter()
			router.POST("/verify/send", controller.SendVerificationCode)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/verify/send", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			verificationRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestVerifyCode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockVerificationRepository, *mocks.MockUserRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful verification",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				"code":  "123456",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				verification := &models.Verification{
					Email:     "test@example.com",
					Code:      "123456",
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}
				verificationRepo.On("FindByEmailAndCode", "test@example.com", "123456").Return(verification, nil)
				userRepo.On("SetUserVerified", "test@example.com").Return(nil)
				verificationRepo.On("DeleteByEmail", "test@example.com").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Verification successful",
		},
		{
			name: "invalid verification code",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				"code":  "wrong123",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				verificationRepo.On("FindByEmailAndCode", "test@example.com", "wrong123").Return(nil, errors.New("verification not found"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Invalid or expired verification code",
		},
		{
			name: "database error when setting user verified",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				"code":  "123456",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				verification := &models.Verification{
					Email:     "test@example.com",
					Code:      "123456",
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}
				verificationRepo.On("FindByEmailAndCode", "test@example.com", "123456").Return(verification, nil)
				userRepo.On("SetUserVerified", "test@example.com").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to verify user",
		},
		{
			name: "invalid request format",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				// Missing code field
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
				"code":  "123456",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, verificationRepo, userRepo := setupVerificationControllerWithMocks()
			tt.setupMocks(verificationRepo, userRepo)

			router := setupVerificationTestRouter()
			router.POST("/verify", controller.VerifyCode)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			verificationRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestResendVerificationCode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockVerificationRepository, *mocks.MockUserRepository)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful resend verification code",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				user := &models.User{
					ID:    1,
					Email: "test@example.com",
				}
				userRepo.On("GetUserByEmail", "test@example.com").Return(user, nil)
				verificationRepo.On("DeleteByEmail", "test@example.com").Return(nil)
				verificationRepo.On("CreateVerification", mock.AnythingOfType("*models.Verification")).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Verification code sent successfully",
		},
		{
			name: "user not found for resend",
			requestBody: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			setupMocks: func(verificationRepo *mocks.MockVerificationRepository, userRepo *mocks.MockUserRepository) {
				userRepo.On("GetUserByEmail", "nonexistent@example.com").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, verificationRepo, userRepo := setupVerificationControllerWithMocks()
			tt.setupMocks(verificationRepo, userRepo)

			router := setupVerificationTestRouter()
			router.POST("/verify/resend", controller.ResendVerificationCode)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/verify/resend", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			verificationRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

// Benchmark tests
func BenchmarkSendVerificationCode(b *testing.B) {
	controller, verificationRepo, userRepo := setupVerificationControllerWithMocks()

	user := &models.User{
		ID:    1,
		Email: "test@example.com",
	}
	userRepo.On("GetUserByEmail", "test@example.com").Return(user, nil)
	verificationRepo.On("DeleteByEmail", "test@example.com").Return(nil)
	verificationRepo.On("CreateVerification", mock.AnythingOfType("*models.Verification")).Return(nil)

	router := setupVerificationTestRouter()
	router.POST("/verify/send", controller.SendVerificationCode)

	requestBody := map[string]interface{}{
		"email": "test@example.com",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/verify/send", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}