package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"strconv"

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
// @Description Create an activity with the provided data
// @Tags activity
// @Accept json
// @Produce json
// @Param activity body models.Activity true "Activity data"
// @Success 201 {object} map[string]interface{} "Activity created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 500 {object} map[string]interface{} "Failed to create activity"
// @Router /activity [post]
func (ac *ActivityController) CreateActivity(c *gin.Context) {
	var activity models.Activity

	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
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

// GetActivitiesByUserID godoc
// @Summary Get all activities for a user
// @Description Retrieve all activities associated with a specific user ID
// @Tags activity
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Activities retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve activities"
// @Router /activity/user/{user_id} [get]
func (ac *ActivityController) GetActivitiesByUserID(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	activities, err := ac.repo.FindAllByUserID(uint(userID))
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
		"data":    activities,
	})
}

// GetActivityByID godoc
// @Summary Get an activity by ID
// @Description Retrieve activity information by activity ID
// @Tags activity
// @Produce json
// @Param id path int true "Activity ID"
// @Success 200 {object} map[string]interface{} "Activity retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity ID"
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

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity retrieved successfully",
		"data":    activity,
	})
}

// UpdateActivity godoc
// @Summary Update an activity
// @Description Update activity information
// @Tags activity
// @Accept json
// @Produce json
// @Param id path int true "Activity ID"
// @Param activity body models.Activity true "Activity data"
// @Success 200 {object} map[string]interface{} "Activity updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
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

	_, err = ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity not found",
			"error":   "No activity exists with the provided ID",
		})
		return
	}

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
// @Description Delete activity by ID
// @Tags activity
// @Produce json
// @Param id path int true "Activity ID"
// @Success 200 {object} map[string]interface{} "Activity deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity ID"
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

	_, err = ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity not found",
			"error":   "No activity exists with the provided ID",
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
