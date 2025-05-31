package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ActivityController struct {
	repo repository.ActivityRepository
}

func NewActivityController(repo repository.ActivityRepository) *ActivityController {
	return &ActivityController{repo: repo}
}

// CreateActivity godoc
// @Summary Create a new activity
// @Description Create an activity with the provided data (requires authentication)
// @Tags activity
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param activity body models.Activity true "Activity data including value field"
// @Success 201 {object} map[string]interface{} "Activity created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to create activity"
// @Router /activity [post]
func (ac *ActivityController) CreateActivity(c *gin.Context) {
	var activity models.Activity

	// Get authenticated user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	activity.UserID = userID.(uint)

	if activity.ActivityType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Activity type is required",
		})
		return
	}

	if err := ac.repo.Create(&activity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create activity",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Activity created successfully",
		"data":    activity,
	})
}

// GetCurrentUserActivities godoc
// @Summary Get activities for current user
// @Description Retrieve all activities for the authenticated user
// @Tags activity
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Activities retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve activities"
// @Router /activity/me [get]
func (ac *ActivityController) GetCurrentUserActivities(c *gin.Context) {
	// Get user ID from the JWT token (set by middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	activities, err := ac.repo.FindAllByUserID(userID.(uint))
	// separate activity by type ["smoke", "workout"]
	groupedActivities := make(map[string]interface{})

	groupedActivities["smoke"] = []interface{}{}
	groupedActivities["workout"] = []interface{}{}

	for _, activity := range activities {
		activityType := activity.ActivityType
		if activityType == "smoke" || activityType == "workout" {
			groupedActivities[activityType] = append(groupedActivities[activityType].([]interface{}), activity)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve activities",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activities retrieved successfully",
		"data":    groupedActivities,
	})
}

// GetActivityByID godoc
// @Summary Get an activity by ID
// @Description Retrieve activity information by activity ID (requires authentication)
// @Tags activity
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Activity ID"
// @Success 200 {object} map[string]interface{} "Activity retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Activity not found"
// @Router /activity/{id} [get]
func (ac *ActivityController) GetActivityByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	activity, err := ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity not found",
			"error":   "No activity exists with the provided ID",
		})
		return
	}

	authenticatedUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if authenticatedUserID.(uint) != activity.UserID {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "You can only access your own activities",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity retrieved successfully",
		"data":    activity,
	})
}

// UpdateActivity godoc
// @Summary Update an activity
// @Description Update activity information including value field (requires authentication)
// @Tags activity
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Activity ID"
// @Param activity body models.Activity true "Activity data including value field"
// @Success 200 {object} map[string]interface{} "Activity updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Activity not found"
// @Failure 500 {object} map[string]interface{} "Failed to update activity"
// @Router /activity/{id} [put]
func (ac *ActivityController) UpdateActivity(c *gin.Context) {
	var activity models.Activity

	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}
	activity.ID = uint(id)

	existingActivity, err := ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity not found",
			"error":   "No activity exists with the provided ID",
		})
		return
	}

	// Check if user is updating their own data
	authenticatedUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if authenticatedUserID.(uint) != existingActivity.UserID {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "You can only update your own activities",
		})
		return
	}

	activity.UserID = existingActivity.UserID

	if err := ac.repo.Update(&activity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update activity",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity updated successfully",
		"data":    activity,
	})
}

// DeleteActivity godoc
// @Summary Delete an activity
// @Description Delete activity by ID (requires authentication)
// @Tags activity
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Activity ID"
// @Success 200 {object} map[string]interface{} "Activity deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Activity not found"
// @Failure 500 {object} map[string]interface{} "Failed to delete activity"
// @Router /activity/{id} [delete]
func (ac *ActivityController) DeleteActivity(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	existingActivity, err := ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity not found",
			"error":   "No activity exists with the provided ID",
		})
		return
	}

	authenticatedUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if authenticatedUserID.(uint) != existingActivity.UserID {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "You can only delete your own activities",
		})
		return
	}

	if err := ac.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete activity",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity deleted successfully",
		"data":    nil,
	})
}

// GetActivitiesByDateRange godoc
// @Summary Get activities by date range
// @Description Retrieve all activities for the authenticated user within a specific date range, grouped by type
// @Tags activity
// @Produce json
// @Security BearerAuth
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{} "Activities retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid date format"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve activities"
// @Router /activity/me/date-range [get]
func (ac *ActivityController) GetActivitiesByDateRange(c *gin.Context) {
	// Get user ID from the JWT token
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

	endDate = endDate.Add(24 * time.Hour)

	activities, err := ac.repo.FindByUserIDAndActivityDateRange(userID.(uint), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve activities",
			"error":   err.Error(),
		})
		return
	}

	groupedActivities := make(map[string]interface{})

	groupedActivities["smoke"] = []interface{}{}
	groupedActivities["workout"] = []interface{}{}

	for _, activity := range activities {
		activityType := activity.ActivityType

		if activityType == "smoke" || activityType == "workout" {
			groupedActivities[activityType] = append(groupedActivities[activityType].([]interface{}), activity)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activities retrieved successfully",
		"data":    groupedActivities,
	})
}

// CountUserActivities
// @Summary Count user activities
// @Description Count the number of activities for the authenticated user
// @Tags activity
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Activity count retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to count activities"
// @Router /activity/me/count [get]
func (ac *ActivityController) CountUserActivities(c *gin.Context) {
	// Get user ID from the JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	count, err := ac.repo.CountUserActivities(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to count activities",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity count retrieved successfully",
		"data":    map[string]int64{"count": count},
	})
}
