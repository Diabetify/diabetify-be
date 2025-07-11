package repository

import (
	"diabetify/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type PredictionJobRepository interface {
	// Basic CRUD operations
	SaveJob(job *models.PredictionJob) error
	GetJobByID(id string) (*models.PredictionJob, error)
	UpdateJob(job *models.PredictionJob) error
	DeleteJob(jobID string) error

	// Status management
	UpdateJobStatus(jobID, status string, errorMessage *string) error
	UpdateJobProgress(jobID string, progress int, step string) error
	UpdateJobStatusWithResult(jobID, status string, predictionID uint) error

	// Query operations
	GetJobsByUserID(userID uint, limit int) ([]*models.PredictionJob, error)
	GetJobsByUserIDAndStatus(userID uint, status string, limit int) ([]*models.PredictionJob, error)
	GetJobsByStatus(status string, limit int) ([]*models.PredictionJob, error)
	GetPendingJobs(limit int) ([]*models.PredictionJob, error)
	GetJobsByDateRange(userID uint, startDate, endDate time.Time) ([]*models.PredictionJob, error)

	// Utility operations
	CancelJob(jobID string) error
	GetActiveJobsCount(userID uint) (int64, error)
	CleanupOldJobs(olderThan time.Time) error
}

type predictionJobRepository struct {
	db *gorm.DB
}

func NewPredictionJobRepository(db *gorm.DB) PredictionJobRepository {
	return &predictionJobRepository{db: db}
}

// ========== BASIC CRUD OPERATIONS ==========

func (r *predictionJobRepository) SaveJob(job *models.PredictionJob) error {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.UpdatedAt = time.Now()

	return r.db.Create(job).Error
}

func (r *predictionJobRepository) GetJobByID(id string) (*models.PredictionJob, error) {
	var job models.PredictionJob
	err := r.db.Preload("User").Preload("Prediction").Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *predictionJobRepository) UpdateJob(job *models.PredictionJob) error {
	job.UpdatedAt = time.Now()
	return r.db.Save(job).Error
}

func (r *predictionJobRepository) DeleteJob(jobID string) error {
	return r.db.Where("id = ?", jobID).Delete(&models.PredictionJob{}).Error
}

// ========== STATUS MANAGEMENT ==========

func (r *predictionJobRepository) UpdateJobStatus(jobID, status string, errorMessage *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if errorMessage != nil {
		updates["error_message"] = *errorMessage
	}

	// Set completed_at if job is finished
	if status == models.JobStatusCompleted || status == models.JobStatusFailed || status == models.JobStatusCancelled {
		now := time.Now()
		updates["completed_at"] = &now
	}

	result := r.db.Model(&models.PredictionJob{}).Where("id = ?", jobID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	return nil
}

func (r *predictionJobRepository) UpdateJobProgress(jobID string, progress int, step string) error {
	updates := map[string]interface{}{
		"progress":   progress,
		"step":       step,
		"updated_at": time.Now(),
	}

	// If progress is 100, consider it completed
	if progress >= 100 {
		updates["status"] = models.JobStatusCompleted
		now := time.Now()
		updates["completed_at"] = &now
	}

	result := r.db.Model(&models.PredictionJob{}).Where("id = ?", jobID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	return nil
}

func (r *predictionJobRepository) UpdateJobStatusWithResult(jobID, status string, predictionID uint) error {
	updates := map[string]interface{}{
		"status":        status,
		"prediction_id": predictionID,
		"progress":      100,
		"step":          models.JobStepCompleted,
		"updated_at":    time.Now(),
	}

	if status == models.JobStatusCompleted {
		now := time.Now()
		updates["completed_at"] = &now
	}

	result := r.db.Model(&models.PredictionJob{}).Where("id = ?", jobID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("job with ID %s not found", jobID)
	}

	return nil
}

// ========== QUERY OPERATIONS ==========

func (r *predictionJobRepository) GetJobsByUserID(userID uint, limit int) ([]*models.PredictionJob, error) {
	var jobs []*models.PredictionJob
	query := r.db.Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Preload("Prediction").Find(&jobs).Error
	return jobs, err
}

func (r *predictionJobRepository) GetJobsByUserIDAndStatus(userID uint, status string, limit int) ([]*models.PredictionJob, error) {
	var jobs []*models.PredictionJob
	query := r.db.Where("user_id = ? AND status = ?", userID, status).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Preload("Prediction").Find(&jobs).Error
	return jobs, err
}

func (r *predictionJobRepository) GetJobsByStatus(status string, limit int) ([]*models.PredictionJob, error) {
	var jobs []*models.PredictionJob
	query := r.db.Where("status = ?", status).
		Order("created_at ASC") // Oldest first for processing

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Preload("User").Find(&jobs).Error
	return jobs, err
}

func (r *predictionJobRepository) GetPendingJobs(limit int) ([]*models.PredictionJob, error) {
	return r.GetJobsByStatus(models.JobStatusPending, limit)
}

func (r *predictionJobRepository) GetJobsByDateRange(userID uint, startDate, endDate time.Time) ([]*models.PredictionJob, error) {
	var jobs []*models.PredictionJob
	err := r.db.Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startDate, endDate).
		Order("created_at DESC").
		Preload("Prediction").
		Find(&jobs).Error
	return jobs, err
}

// ========== UTILITY OPERATIONS ==========

func (r *predictionJobRepository) CancelJob(jobID string) error {
	// Only allow cancellation of pending or processing jobs
	result := r.db.Model(&models.PredictionJob{}).
		Where("id = ? AND status IN (?)", jobID, []string{models.JobStatusPending, models.JobStatusProcessing}).
		Updates(map[string]interface{}{
			"status":       models.JobStatusCancelled,
			"updated_at":   time.Now(),
			"completed_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("job not found or cannot be cancelled (current status may not allow cancellation)")
	}

	return nil
}

func (r *predictionJobRepository) GetActiveJobsCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.PredictionJob{}).
		Where("user_id = ? AND status IN (?)", userID, []string{models.JobStatusPending, models.JobStatusProcessing}).
		Count(&count).Error
	return count, err
}

func (r *predictionJobRepository) CleanupOldJobs(olderThan time.Time) error {
	// Delete completed/failed/cancelled jobs older than specified time
	result := r.db.Where("completed_at < ? AND status IN (?)",
		olderThan,
		[]string{models.JobStatusCompleted, models.JobStatusFailed, models.JobStatusCancelled},
	).Delete(&models.PredictionJob{})

	if result.Error != nil {
		return result.Error
	}

	// Log cleanup result
	if result.RowsAffected > 0 {
		fmt.Printf("Cleaned up %d old jobs\n", result.RowsAffected)
	}

	return nil
}

// ========== ADDITIONAL HELPER METHODS ==========

// GetJobStatistics returns job statistics for a user
func (r *predictionJobRepository) GetJobStatistics(userID uint) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count jobs by status
	statuses := []string{
		models.JobStatusPending,
		models.JobStatusProcessing,
		models.JobStatusCompleted,
		models.JobStatusFailed,
		models.JobStatusCancelled,
	}

	for _, status := range statuses {
		var count int64
		err := r.db.Model(&models.PredictionJob{}).
			Where("user_id = ? AND status = ?", userID, status).
			Count(&count).Error
		if err != nil {
			return nil, err
		}
		stats[status] = count
	}

	// Total jobs
	var totalCount int64
	err := r.db.Model(&models.PredictionJob{}).
		Where("user_id = ?", userID).
		Count(&totalCount).Error
	if err != nil {
		return nil, err
	}
	stats["total"] = totalCount

	return stats, nil
}

// IsJobOwnedByUser checks if a job belongs to a specific user
func (r *predictionJobRepository) IsJobOwnedByUser(jobID string, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.PredictionJob{}).
		Where("id = ? AND user_id = ?", jobID, userID).
		Count(&count).Error
	return count > 0, err
}
