package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RecommendationController struct {
	repo repository.RecommendationRepository
}

func NewRecommendationController(repo repository.RecommendationRepository) *RecommendationController {
	return &RecommendationController{repo: repo}
}

// CreateRecommendation godoc
// @Summary Create a new recommendation
// @Description Create a recommendation with the provided data
// @Tags recommendations
// @Accept json
// @Produce json
// @Param recommendation body models.Recommendation true "Recommendation data"
// @Success 201 {object} map[string]interface{} "Recommendation created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 500 {object} map[string]interface{} "Failed to create recommendation"
// @Router /recommendations [post]
func (rc *RecommendationController) CreateRecommendation(c *gin.Context) {
	var recommendation models.Recommendation

	if err := c.ShouldBindJSON(&recommendation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	if err := rc.repo.Create(&recommendation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create recommendation",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Recommendation created successfully",
		"data":    recommendation,
	})
}

// GetRecommendationsByUserID godoc
// @Summary Get all recommendations for a user
// @Description Retrieve all recommendations associated with a specific user ID
// @Tags recommendations
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} map[string]interface{} "Recommendations retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve recommendations"
// @Router /recommendations/user/{user_id} [get]
func (rc *RecommendationController) GetRecommendationsByUserID(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	recommendations, err := rc.repo.FindAllByUserID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve recommendations",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Recommendations retrieved successfully",
		"data":    recommendations,
	})
}

// GetRecommendationByID godoc
// @Summary Get a recommendation by ID
// @Description Retrieve recommendation information by recommendation ID
// @Tags recommendations
// @Produce json
// @Param id path int true "Recommendation ID"
// @Success 200 {object} map[string]interface{} "Recommendation retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid recommendation ID"
// @Failure 404 {object} map[string]interface{} "Recommendation not found"
// @Router /recommendations/{id} [get]
func (rc *RecommendationController) GetRecommendationByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid recommendation ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	recommendation, err := rc.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Recommendation not found",
			"error":   "No recommendation exists with the provided ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Recommendation retrieved successfully",
		"data":    recommendation,
	})
}

// UpdateRecommendation godoc
// @Summary Update a recommendation
// @Description Update recommendation information
// @Tags recommendations
// @Accept json
// @Produce json
// @Param id path int true "Recommendation ID"
// @Param recommendation body models.Recommendation true "Recommendation data"
// @Success 200 {object} map[string]interface{} "Recommendation updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 404 {object} map[string]interface{} "Recommendation not found"
// @Failure 500 {object} map[string]interface{} "Failed to update recommendation"
// @Router /recommendations/{id} [put]
func (rc *RecommendationController) UpdateRecommendation(c *gin.Context) {
	var recommendation models.Recommendation

	if err := c.ShouldBindJSON(&recommendation); err != nil {
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
			"message": "Invalid recommendation ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}
	recommendation.ID = uint(id)

	_, err = rc.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Recommendation not found",
			"error":   "No recommendation exists with the provided ID",
		})
		return
	}

	if err := rc.repo.Update(&recommendation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update recommendation",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Recommendation updated successfully",
		"data":    recommendation,
	})
}

// DeleteRecommendation godoc
// @Summary Delete a recommendation
// @Description Delete recommendation by ID
// @Tags recommendations
// @Produce json
// @Param id path int true "Recommendation ID"
// @Success 200 {object} map[string]interface{} "Recommendation deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid recommendation ID"
// @Failure 404 {object} map[string]interface{} "Recommendation not found"
// @Failure 500 {object} map[string]interface{} "Failed to delete recommendation"
// @Router /recommendations/{id} [delete]
func (rc *RecommendationController) DeleteRecommendation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid recommendation ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}
	_, err = rc.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Recommendation not found",
			"error":   "No recommendation exists with the provided ID",
		})
		return
	}

	if err := rc.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete recommendation",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Recommendation deleted successfully",
		"data":    nil,
	})
}
