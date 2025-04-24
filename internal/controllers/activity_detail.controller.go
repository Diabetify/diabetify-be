package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ActivityDetailController struct {
	repo repository.ActivityDetailRepository
}

func NewActivityDetailController(repo repository.ActivityDetailRepository) *ActivityDetailController {
	return &ActivityDetailController{repo: repo}
}

// CreateActivityDetail godoc
// @Summary Create a new activity detail
// @Description Create an activity detail with the provided data
// @Tags activity-detail
// @Accept json
// @Produce json
// @Param activityDetail body models.ActivityDetail true "Activity Detail data"
// @Success 201 {object} map[string]interface{} "Activity detail created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 500 {object} map[string]interface{} "Failed to create activity detail"
// @Router /activity-detail [post]
func (adc *ActivityDetailController) CreateActivityDetail(c *gin.Context) {
	var activityDetail models.ActivityDetail

	if err := c.ShouldBindJSON(&activityDetail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	if err := adc.repo.Create(&activityDetail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create activity detail",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Activity detail created successfully",
		"data":    activityDetail,
	})
}

// GetActivityDetailsByActivityID godoc
// @Summary Get all details for an activity
// @Description Retrieve all activity details associated with a specific activity ID
// @Tags activity-detail
// @Produce json
// @Param activity_id path int true "Activity ID"
// @Success 200 {object} map[string]interface{} "Activity details retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity ID"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve activity details"
// @Router /activity-detail/activity/{activity_id} [get]
func (adc *ActivityDetailController) GetActivityDetailsByActivityID(c *gin.Context) {
	activityID, err := strconv.ParseUint(c.Param("activity_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	details, err := adc.repo.FindByActivityID(uint(activityID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve activity details",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity details retrieved successfully",
		"data":    details,
	})
}

// GetActivityDetailByID godoc
// @Summary Get an activity detail by ID
// @Description Retrieve activity detail information by detail ID
// @Tags activity-detail
// @Produce json
// @Param id path int true "Activity Detail ID"
// @Success 200 {object} map[string]interface{} "Activity detail retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity detail ID"
// @Failure 404 {object} map[string]interface{} "Activity detail not found"
// @Router /activity-detail/{id} [get]
func (adc *ActivityDetailController) GetActivityDetailByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity detail ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	detail, err := adc.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity detail not found",
			"error":   "No activity detail exists with the provided ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity detail retrieved successfully",
		"data":    detail,
	})
}

// UpdateActivityDetail godoc
// @Summary Update an activity detail
// @Description Update activity detail information
// @Tags activity-detail
// @Accept json
// @Produce json
// @Param id path int true "Activity Detail ID"
// @Param activityDetail body models.ActivityDetail true "Activity Detail data"
// @Success 200 {object} map[string]interface{} "Activity detail updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 404 {object} map[string]interface{} "Activity detail not found"
// @Failure 500 {object} map[string]interface{} "Failed to update activity detail"
// @Router /activity-detail/{id} [put]
func (adc *ActivityDetailController) UpdateActivityDetail(c *gin.Context) {
	var activityDetail models.ActivityDetail

	if err := c.ShouldBindJSON(&activityDetail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Ensure the ID in the path matches the ID in the body
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity detail ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}
	activityDetail.ID = uint(id)

	_, err = adc.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity detail not found",
			"error":   "No activity detail exists with the provided ID",
		})
		return
	}

	if err := adc.repo.Update(&activityDetail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update activity detail",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity detail updated successfully",
		"data":    activityDetail,
	})
}

// DeleteActivityDetail godoc
// @Summary Delete an activity detail
// @Description Delete activity detail by ID
// @Tags activity-detail
// @Produce json
// @Param id path int true "Activity Detail ID"
// @Success 200 {object} map[string]interface{} "Activity detail deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity detail ID"
// @Failure 404 {object} map[string]interface{} "Activity detail not found"
// @Failure 500 {object} map[string]interface{} "Failed to delete activity detail"
// @Router /activity-detail/{id} [delete]
func (adc *ActivityDetailController) DeleteActivityDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity detail ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	_, err = adc.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Activity detail not found",
			"error":   "No activity detail exists with the provided ID",
		})
		return
	}

	if err := adc.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete activity detail",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity detail deleted successfully",
		"data":    nil,
	})
}

// DeleteActivityDetailsByActivityID godoc
// @Summary Delete all details for an activity
// @Description Delete all activity details associated with a specific activity ID
// @Tags activity-detail
// @Produce json
// @Param activity_id path int true "Activity ID"
// @Success 200 {object} map[string]interface{} "Activity details deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid activity ID"
// @Failure 500 {object} map[string]interface{} "Failed to delete activity details"
// @Router /activity-detail/activity/{activity_id} [delete]
func (adc *ActivityDetailController) DeleteActivityDetailsByActivityID(c *gin.Context) {
	activityID, err := strconv.ParseUint(c.Param("activity_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid activity ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	if err := adc.repo.DeleteByActivityID(uint(activityID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete activity details",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Activity details deleted successfully",
		"data":    nil,
	})
}
