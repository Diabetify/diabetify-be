package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserProfileController struct {
	repo repository.UserProfileRepository
}

func NewUserProfileController(repo repository.UserProfileRepository) *UserProfileController {
	return &UserProfileController{repo: repo}
}

// GetUserProfile godoc
// @Summary Get user profile
// @Description Retrieve the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User profile retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /profile [get]
func (pc *UserProfileController) GetUserProfile(c *gin.Context) {
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

	// Find the profile
	profile, err := pc.repo.FindByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Profile not found",
			"error":   "No profile exists for this user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User profile retrieved successfully",
		"data":    profile,
	})
}

// CreateUserProfile godoc
// @Summary Create user profile
// @Description Create a profile for the authenticated user
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body models.UserProfile true "Profile data"
// @Success 201 {object} map[string]interface{} "Profile created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to create profile"
// @Router /profile [post]
func (pc *UserProfileController) CreateUserProfile(c *gin.Context) {
	var profile models.UserProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

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

	profile.UserID = userID.(uint)

	if profile.Weight != nil && profile.Height != nil && *profile.Height > 0 {
		heightInMeters := float64(*profile.Height) / 100.0
		bmi := float64(*profile.Weight) / (heightInMeters * heightInMeters)
		bmi = math.Round(bmi*10) / 10
		profile.BMI = &bmi
	}

	if err := pc.repo.Create(&profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create profile",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Profile created successfully",
		"data":    profile,
	})
}

// UpdateUserProfile godoc
// @Summary Update user profile
// @Description Update the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body models.UserProfile true "Profile data"
// @Success 200 {object} map[string]interface{} "Profile updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 500 {object} map[string]interface{} "Failed to update profile"
// @Router /profile [put]
func (pc *UserProfileController) UpdateUserProfile(c *gin.Context) {
	var updatedProfile models.UserProfile
	if err := c.ShouldBindJSON(&updatedProfile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

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

	// Get existing profile
	existingProfile, err := pc.repo.FindByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Profile not found",
			"error":   "No profile exists for this user",
		})
		return
	}

	// Update fields
	updatedProfile.ID = existingProfile.ID
	updatedProfile.UserID = userID.(uint)

	// Recalculate BMI if weight and height are provided
	if updatedProfile.Weight != nil && updatedProfile.Height != nil && *updatedProfile.Height > 0 {
		heightInMeters := float64(*updatedProfile.Height) / 100.0
		bmi := float64(*updatedProfile.Weight) / (heightInMeters * heightInMeters)
		bmi = math.Round(bmi*10) / 10 // Round to 1 decimal place
		updatedProfile.BMI = &bmi
	}

	if err := pc.repo.Update(&updatedProfile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update profile",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile updated successfully",
		"data":    updatedProfile,
	})
}

// DeleteUserProfile godoc
// @Summary Delete user profile
// @Description Delete the authenticated user's profile
// @Tags profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Profile deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to delete profile"
// @Router /profile [delete]
func (pc *UserProfileController) DeleteUserProfile(c *gin.Context) {
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

	if err := pc.repo.DeleteByUserID(userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete profile",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile deleted successfully",
		"data":    nil,
	})
}

// PatchUserProfile godoc
// @Summary Patch user profile
// @Description Update specific fields of the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body map[string]interface{} true "Profile data to update"
// @Success 200 {object} map[string]interface{} "Profile patched successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 500 {object} map[string]interface{} "Failed to update profile"
// @Router /profile [patch]
func (pc *UserProfileController) PatchUserProfile(c *gin.Context) {
	var patchData map[string]interface{}
	if err := c.ShouldBindJSON(&patchData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	existingProfile, err := pc.repo.FindByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Profile not found",
			"error":   "No profile exists for this user",
		})
		return
	}

	recalculateBMI := false
	var weight float64
	var height float64

	if weightVal, ok := patchData["weight"]; ok {
		if w, ok := weightVal.(float64); ok {
			weight = w
			if existingProfile.Height != nil {
				height = float64(*existingProfile.Height)
				recalculateBMI = true
			}
		}
	} else if existingProfile.Weight != nil {
		weight = float64(*existingProfile.Weight)
	}

	if heightVal, ok := patchData["height"]; ok {
		if h, ok := heightVal.(float64); ok {
			height = h
			if existingProfile.Weight != nil || patchData["weight"] != nil {
				if !recalculateBMI {
					weight = float64(*existingProfile.Weight)
				}
				recalculateBMI = true
			}
		}
	} else if existingProfile.Height != nil {
		height = float64(*existingProfile.Height)
	}

	if recalculateBMI && height > 0 {
		heightInMeters := height / 100.0
		bmi := weight / (heightInMeters * heightInMeters)
		bmi = math.Round(bmi*10) / 10
		bmiValue := float64(bmi)
		patchData["bmi"] = bmiValue
	}

	if err := pc.repo.Patch(userID.(uint), patchData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update profile",
			"error":   err.Error(),
		})
		return
	}

	updatedProfile, err := pc.repo.FindByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve updated profile",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile patched successfully",
		"data":    updatedProfile,
	})
}
