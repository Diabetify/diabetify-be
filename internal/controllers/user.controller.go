package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	repo *repository.UserRepository
}

func NewUserController(repo *repository.UserRepository) *UserController {
	return &UserController{repo: repo}
}
func (uc *UserController) CreateUser(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	user.Verified = false

	if err := uc.repo.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create user",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "User registered. Please verify your email.",
		"data":    nil,
	})
}

func (uc *UserController) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")

	user, err := uc.repo.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user associated with this email",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User retrieved successfully",
		"data":    user,
	})
}

func (uc *UserController) GetUserByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	user, err := uc.repo.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with the provided ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User retrieved successfully",
		"data":    user,
	})
}

func (uc *UserController) UpdateUser(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	if err := uc.repo.UpdateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update user",
			"error":   "Database update failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User updated successfully",
		"data":    user,
	})
}

func (uc *UserController) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	if err := uc.repo.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete user",
			"error":   "Database deletion failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User deleted successfully",
		"data":    nil,
	})
}
