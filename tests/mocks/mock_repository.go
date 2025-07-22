package mocks

import (
	"context"
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"time"

	"github.com/stretchr/testify/mock"
)

// Shared MockActivityRepository
type MockActivityRepository struct {
	mock.Mock
}

func (m *MockActivityRepository) Create(activity *models.Activity) error {
	args := m.Called(activity)
	return args.Error(0)
}

func (m *MockActivityRepository) FindAllByUserID(userID uint, limit int) ([]models.Activity, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]models.Activity), args.Error(1)
}

func (m *MockActivityRepository) FindByID(id uint) (*models.Activity, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Activity), args.Error(1)
}

func (m *MockActivityRepository) Update(activity *models.Activity) error {
	args := m.Called(activity)
	return args.Error(0)
}

func (m *MockActivityRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockActivityRepository) FindByUserIDAndActivityDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error) {
	args := m.Called(userID, startDate, endDate)
	return args.Get(0).([]models.Activity), args.Error(1)
}

func (m *MockActivityRepository) GetActivitiesByUserIDAndTypeAndDateRange(userID uint, activityType string, startDate, endDate time.Time) ([]models.Activity, error) {
	args := m.Called(userID, activityType, startDate, endDate)
	return args.Get(0).([]models.Activity), args.Error(1)
}

func (m *MockActivityRepository) GetActivitiesByUserIDAndType(userID uint, activityType string) ([]models.Activity, error) {
	args := m.Called(userID, activityType)
	return args.Get(0).([]models.Activity), args.Error(1)
}

func (m *MockActivityRepository) CountUserActivities(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

// Shared MockUserProfileRepository
type MockUserProfileRepository struct {
	mock.Mock
}

func (m *MockUserProfileRepository) Create(profile *models.UserProfile) error {
	args := m.Called(profile)
	return args.Error(0)
}

func (m *MockUserProfileRepository) FindByID(id uint) (*models.UserProfile, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockUserProfileRepository) FindByUserID(userID uint) (*models.UserProfile, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockUserProfileRepository) Update(profile *models.UserProfile) error {
	args := m.Called(profile)
	return args.Error(0)
}

func (m *MockUserProfileRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserProfileRepository) DeleteByUserID(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserProfileRepository) Patch(userID uint, data map[string]interface{}) error {
	args := m.Called(userID, data)
	return args.Error(0)
}

// Shared MockUserRepository - implements the same methods as repository.UserRepository struct
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) PatchUser(id uint, data map[string]interface{}) error {
	args := m.Called(id, data)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) SetUserVerified(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockUserRepository) IsUserVerified(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdateLastPredictionTime(userID uint, timestamp *time.Time) error {
	args := m.Called(userID, timestamp)
	return args.Error(0)
}

// Shared MockPredictionRepository
type MockPredictionRepository struct {
	mock.Mock
}

func (m *MockPredictionRepository) SavePrediction(prediction *models.Prediction) error {
	args := m.Called(prediction)
	return args.Error(0)
}

func (m *MockPredictionRepository) GetPredictionsByUserID(userID uint, limit int) ([]models.Prediction, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]models.Prediction), args.Error(1)
}

func (m *MockPredictionRepository) GetPredictionsByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]models.Prediction, error) {
	args := m.Called(userID, startDate, endDate)
	return args.Get(0).([]models.Prediction), args.Error(1)
}

func (m *MockPredictionRepository) GetPredictionByID(id uint) (*models.Prediction, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Prediction), args.Error(1)
}

func (m *MockPredictionRepository) DeletePrediction(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockPredictionRepository) GetPredictionScoreByUserIDAndDateRange(userID uint, startDate, endDate time.Time) ([]repository.PredictionScore, error) {
	args := m.Called(userID, startDate, endDate)
	return args.Get(0).([]repository.PredictionScore), args.Error(1)
}

func (m *MockPredictionRepository) GetLatestPredictionByUserID(userID uint) (*models.Prediction, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Prediction), args.Error(1)
}

func (m *MockPredictionRepository) UpdatePrediction(prediction *models.Prediction) error {
	args := m.Called(prediction)
	return args.Error(0)
}

// Shared MockMLClient
type MockMLClient struct {
	mock.Mock
}

func (m *MockMLClient) Predict(ctx context.Context, features []float64) (*models.PredictionResponse, error) {
	args := m.Called(ctx, features)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PredictionResponse), args.Error(1)
}

func (m *MockMLClient) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMLClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockResetPasswordRepository is a mock implementation of ResetPasswordRepository
type MockResetPasswordRepository struct {
	mock.Mock
}

func (m *MockResetPasswordRepository) CreateResetPassword(resetPassword *models.ResetPassword) error {
	args := m.Called(resetPassword)
	return args.Error(0)
}

func (m *MockResetPasswordRepository) FindByEmailAndCode(email, code string) (*models.ResetPassword, error) {
	args := m.Called(email, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ResetPassword), args.Error(1)
}

func (m *MockResetPasswordRepository) DeleteByEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

// MockVerificationRepository is a mock implementation of VerificationRepository
type MockVerificationRepository struct {
	mock.Mock
}

func (m *MockVerificationRepository) CreateVerification(verification *models.Verification) error {
	args := m.Called(verification)
	return args.Error(0)
}

func (m *MockVerificationRepository) FindByEmail(email string) (*models.Verification, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Verification), args.Error(1)
}

func (m *MockVerificationRepository) FindByEmailAndCode(email, code string) (*models.Verification, error) {
	args := m.Called(email, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Verification), args.Error(1)
}

func (m *MockVerificationRepository) DeleteByEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

type MockPredictionJobRepository struct {
	mock.Mock
}

// Basic CRUD operations
func (m *MockPredictionJobRepository) SaveJob(job *models.PredictionJob) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockPredictionJobRepository) GetJobByID(id string) (*models.PredictionJob, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PredictionJob), args.Error(1)
}

func (m *MockPredictionJobRepository) UpdateJob(job *models.PredictionJob) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockPredictionJobRepository) DeleteJob(jobID string) error {
	args := m.Called(jobID)
	return args.Error(0)
}

// Status management
func (m *MockPredictionJobRepository) UpdateJobStatus(jobID, status string, errorMessage *string) error {
	args := m.Called(jobID, status, errorMessage)
	return args.Error(0)
}

func (m *MockPredictionJobRepository) UpdateJobStatusWithResult(jobID, status string, predictionID uint) error {
	args := m.Called(jobID, status, predictionID)
	return args.Error(0)
}

// Query operations
func (m *MockPredictionJobRepository) GetJobsByUserID(userID uint, limit int) ([]*models.PredictionJob, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]*models.PredictionJob), args.Error(1)
}

func (m *MockPredictionJobRepository) GetJobsByUserIDAndStatus(userID uint, status string, limit int) ([]*models.PredictionJob, error) {
	args := m.Called(userID, status, limit)
	return args.Get(0).([]*models.PredictionJob), args.Error(1)
}

func (m *MockPredictionJobRepository) GetJobsByStatus(status string, limit int) ([]*models.PredictionJob, error) {
	args := m.Called(status, limit)
	return args.Get(0).([]*models.PredictionJob), args.Error(1)
}

func (m *MockPredictionJobRepository) GetPendingJobs(limit int) ([]*models.PredictionJob, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.PredictionJob), args.Error(1)
}

func (m *MockPredictionJobRepository) GetJobsByDateRange(userID uint, startDate, endDate time.Time) ([]*models.PredictionJob, error) {
	args := m.Called(userID, startDate, endDate)
	return args.Get(0).([]*models.PredictionJob), args.Error(1)
}

// Utility operations
func (m *MockPredictionJobRepository) CancelJob(jobID string) error {
	args := m.Called(jobID)
	return args.Error(0)
}

func (m *MockPredictionJobRepository) GetActiveJobsCount(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPredictionJobRepository) CleanupOldJobs(olderThan time.Time) error {
	args := m.Called(olderThan)
	return args.Error(0)
}

// Additional helper methods
func (m *MockPredictionJobRepository) GetJobStatistics(userID uint) (map[string]int64, error) {
	args := m.Called(userID)
	return args.Get(0).(map[string]int64), args.Error(1)
}

func (m *MockPredictionJobRepository) IsJobOwnedByUser(jobID string, userID uint) (bool, error) {
	args := m.Called(jobID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMLClient) PredictAsync(ctx context.Context, jobID string, features []float64) error {
	args := m.Called(ctx, jobID, features)
	return args.Error(0)
}

func (m *MockMLClient) HealthCheckAsync(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockPredictionJobWorker struct {
	mock.Mock
}

func (m *MockPredictionJobWorker) Start() {
	m.Called()
}

func (m *MockPredictionJobWorker) Stop() {
	m.Called()
}

func (m *MockPredictionJobWorker) SubmitJob(jobRequest models.PredictionJobRequest) error {
	args := m.Called(jobRequest)
	return args.Error(0)
}

func (m *MockPredictionJobWorker) GetWhatIfResult(jobID string) (map[string]interface{}, bool, error) {
	args := m.Called(jobID)
	return args.Get(0).(map[string]interface{}), args.Bool(1), args.Error(2)
}

func (m *MockPredictionJobWorker) GetStatus() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}
