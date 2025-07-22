package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"
	"log"
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

	// Additional helper methods
	GetJobStatistics(userID uint) (map[string]int64, error)
	IsJobOwnedByUser(jobID string, userID uint) (bool, error)
}

type predictionJobRepository struct {
	db        *gorm.DB
	useShards bool // Flag to enable/disable sharding
}

func NewPredictionJobRepository(db *gorm.DB) PredictionJobRepository {
	return &predictionJobRepository{
		db:        db,
		useShards: db == nil,
	}
}

// NewShardedPredictionJobRepository creates a prediction job repository that uses sharding
func NewShardedPredictionJobRepository() PredictionJobRepository {
	return &predictionJobRepository{
		db:        nil,
		useShards: true,
	}
}

// ========== BASIC CRUD OPERATIONS ==========

func (r *predictionJobRepository) SaveJob(job *models.PredictionJob) error {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.UpdatedAt = time.Now()

	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			return db.Create(job).Error
		})
	}

	return r.db.Create(job).Error
}

func (r *predictionJobRepository) GetJobByID(id string) (*models.PredictionJob, error) {
	if r.useShards {
		// Since we don't know the user_id from just the job ID, we need to search all shards
		var foundJob *models.PredictionJob

		shards := database.Manager.GetAllShards()
		for shardName, db := range shards {
			var job models.PredictionJob
			err := db.Preload("User").Preload("Prediction").Where("id = ?", id).First(&job).Error
			if err == nil {
				foundJob = &job
				break
			} else if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
			}
		}

		if foundJob == nil {
			return nil, gorm.ErrRecordNotFound
		}

		return foundJob, nil
	}

	var job models.PredictionJob
	err := r.db.Preload("User").Preload("Prediction").Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *predictionJobRepository) UpdateJob(job *models.PredictionJob) error {
	job.UpdatedAt = time.Now()

	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			return db.Save(job).Error
		})
	}

	return r.db.Save(job).Error
}

func (r *predictionJobRepository) DeleteJob(jobID string) error {
	if r.useShards {
		// Since we don't know the user_id, we need to find it first
		job, err := r.GetJobByID(jobID)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			return db.Where("id = ?", jobID).Delete(&models.PredictionJob{}).Error
		})
	}

	return r.db.Where("id = ?", jobID).Delete(&models.PredictionJob{}).Error
}

// ========== STATUS MANAGEMENT ==========

func (r *predictionJobRepository) UpdateJobStatus(jobID, status string, errorMessage *string) error {
	if r.useShards {
		// Since we don't know the user_id, we need to find it first
		job, err := r.GetJobByID(jobID)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			updates := map[string]interface{}{
				"status":     status,
				"updated_at": time.Now(),
			}

			if errorMessage != nil {
				updates["error_message"] = *errorMessage
			}

			// Set completed_at if job is finished
			if status == "completed" || status == "failed" || status == "cancelled" {
				now := time.Now()
				updates["completed_at"] = &now
			}

			result := db.Model(&models.PredictionJob{}).Where("id = ?", jobID).Updates(updates)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return fmt.Errorf("job with ID %s not found", jobID)
			}

			return nil
		})
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if errorMessage != nil {
		updates["error_message"] = *errorMessage
	}

	// Set completed_at if job is finished
	if status == "completed" || status == "failed" || status == "cancelled" {
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
	if r.useShards {
		// Since we don't know the user_id, we need to find it first
		job, err := r.GetJobByID(jobID)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			updates := map[string]interface{}{
				"status":        status,
				"prediction_id": predictionID,
				"updated_at":    time.Now(),
			}

			if status == models.JobStatusCompleted {
				now := time.Now()
				updates["completed_at"] = &now
			}

			result := db.Model(&models.PredictionJob{}).Where("id = ?", jobID).Updates(updates)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return fmt.Errorf("job with ID %s not found", jobID)
			}

			return nil
		})
	}

	updates := map[string]interface{}{
		"status":        status,
		"prediction_id": predictionID,
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
	if r.useShards {
		var jobs []*models.PredictionJob
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			query := db.Where("user_id = ?", userID).
				Order("created_at DESC")

			if limit > 0 {
				query = query.Limit(limit)
			}

			return query.Preload("Prediction").Find(&jobs).Error
		})

		if err != nil {
			log.Printf("Error querying jobs for user %d: %v", userID, err)
		}

		return jobs, err
	}

	var jobs []*models.PredictionJob
	query := r.db.Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Preload("Prediction").Find(&jobs).Error
	if err != nil {
		log.Printf("Error querying jobs for user %d: %v", userID, err)
	}

	return jobs, err
}

func (r *predictionJobRepository) GetJobsByUserIDAndStatus(userID uint, status string, limit int) ([]*models.PredictionJob, error) {
	if r.useShards {
		var jobs []*models.PredictionJob
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			query := db.Where("user_id = ? AND status = ?", userID, status).
				Order("created_at DESC")

			if limit > 0 {
				query = query.Limit(limit)
			}

			return query.Preload("Prediction").Find(&jobs).Error
		})

		if err != nil {
			log.Printf("Error querying jobs for user %d with status %s: %v", userID, status, err)
		}

		return jobs, err
	}

	var jobs []*models.PredictionJob
	query := r.db.Where("user_id = ? AND status = ?", userID, status).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Preload("Prediction").Find(&jobs).Error
	if err != nil {
		log.Printf("Error querying jobs for user %d with status %s: %v", userID, status, err)
	}

	return jobs, err
}

func (r *predictionJobRepository) GetJobsByStatus(status string, limit int) ([]*models.PredictionJob, error) {
	if r.useShards {
		// For global status queries across all users, we need to query all shards
		var allJobs []*models.PredictionJob

		shards := database.Manager.GetAllShards()
		for shardName, db := range shards {
			var jobs []*models.PredictionJob
			query := db.Where("status = ?", status).
				Order("created_at ASC") // Oldest first for processing

			// Apply limit per shard and let the caller handle final limiting
			err := query.Preload("User").Find(&jobs).Error
			if err != nil {
				log.Printf("Error querying jobs from shard %s: %v", shardName, err)
				return nil, fmt.Errorf("error searching shard %s: %v", shardName, err)
			}
			allJobs = append(allJobs, jobs...)
		}

		// Sort all jobs by created_at and apply limit
		// Note: This is a simple implementation. For better performance with large datasets,
		// consider implementing a more sophisticated approach
		if len(allJobs) > 1 {
			// Simple bubble sort by created_at (for production, use a more efficient sorting algorithm)
			for i := 0; i < len(allJobs)-1; i++ {
				for j := 0; j < len(allJobs)-i-1; j++ {
					if allJobs[j].CreatedAt.After(allJobs[j+1].CreatedAt) {
						allJobs[j], allJobs[j+1] = allJobs[j+1], allJobs[j]
					}
				}
			}
		}

		// Apply limit
		if limit > 0 && len(allJobs) > limit {
			allJobs = allJobs[:limit]
		}

		return allJobs, nil
	}

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
	if r.useShards {
		var jobs []*models.PredictionJob
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startDate, endDate).
				Order("created_at DESC").
				Preload("Prediction").
				Find(&jobs).Error
		})

		if err != nil {
			log.Printf("Error querying jobs by date range for user %d: %v", userID, err)
		}

		return jobs, err
	}

	var jobs []*models.PredictionJob
	err := r.db.Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startDate, endDate).
		Order("created_at DESC").
		Preload("Prediction").
		Find(&jobs).Error

	if err != nil {
		log.Printf("Error querying jobs by date range for user %d: %v", userID, err)
	}

	return jobs, err
}

// ========== UTILITY OPERATIONS ==========

func (r *predictionJobRepository) CancelJob(jobID string) error {
	if r.useShards {
		// Since we don't know the user_id, we need to find it first
		job, err := r.GetJobByID(jobID)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			// Only allow cancellation of pending or processing jobs
			result := db.Model(&models.PredictionJob{}).
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
		})
	}

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
	if r.useShards {
		var count int64
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Model(&models.PredictionJob{}).
				Where("user_id = ? AND status IN (?)", userID, []string{models.JobStatusPending, models.JobStatusProcessing}).
				Count(&count).Error
		})

		if err != nil {
			log.Printf("Error counting active jobs for user %d: %v", userID, err)
		}

		return count, err
	}

	var count int64
	err := r.db.Model(&models.PredictionJob{}).
		Where("user_id = ? AND status IN (?)", userID, []string{models.JobStatusPending, models.JobStatusProcessing}).
		Count(&count).Error

	if err != nil {
		log.Printf("Error counting active jobs for user %d: %v", userID, err)
	}

	return count, err
}

func (r *predictionJobRepository) CleanupOldJobs(olderThan time.Time) error {
	if r.useShards {
		// For cleanup operations across all users, we need to process all shards
		shards := database.Manager.GetAllShards()
		totalDeleted := int64(0)

		for shardName, db := range shards {
			err := func() error {
				result := db.Where("completed_at < ? AND status IN (?)",
					olderThan,
					[]string{models.JobStatusCompleted, models.JobStatusFailed, models.JobStatusCancelled},
				).Delete(&models.PredictionJob{})

				if result.Error != nil {
					return fmt.Errorf("error cleaning up shard %s: %v", shardName, result.Error)
				}

				totalDeleted += result.RowsAffected
				return nil
			}()

			if err != nil {
				log.Printf("Error during cleanup of shard %s: %v", shardName, err)
				return err
			}
		}

		// Log cleanup result
		if totalDeleted > 0 {
			fmt.Printf("Cleaned up %d old jobs across all shards\n", totalDeleted)
		}

		return nil
	}

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
	if r.useShards {
		stats := make(map[string]int64)

		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
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
				err := db.Model(&models.PredictionJob{}).
					Where("user_id = ? AND status = ?", userID, status).
					Count(&count).Error
				if err != nil {
					return err
				}
				stats[status] = count
			}

			// Total jobs
			var totalCount int64
			err := db.Model(&models.PredictionJob{}).
				Where("user_id = ?", userID).
				Count(&totalCount).Error
			if err != nil {
				return err
			}
			stats["total"] = totalCount

			return nil
		})

		return stats, err
	}

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
	if r.useShards {
		var count int64
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Model(&models.PredictionJob{}).
				Where("id = ? AND user_id = ?", jobID, userID).
				Count(&count).Error
		})
		return count > 0, err
	}

	var count int64
	err := r.db.Model(&models.PredictionJob{}).
		Where("id = ? AND user_id = ?", jobID, userID).
		Count(&count).Error
	return count > 0, err
}
