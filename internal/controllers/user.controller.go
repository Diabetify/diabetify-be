package controllers

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"diabetify/internal/utils"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserController struct {
	repo    *repository.UserRepository
	rp_repo *repository.ResetPasswordRepository
}

func NewUserController(repo *repository.UserRepository, rp_repo *repository.ResetPasswordRepository) *UserController {
	return &UserController{repo: repo, rp_repo: rp_repo}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 8)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	// SHA256
	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return hex.EncodeToString(salt) + hex.EncodeToString(hash), nil
}

// Verify password
func verifyPassword(hashedPassword, password string) bool {
	if len(hashedPassword) < 16 {
		return false
	}

	salt, err := hex.DecodeString(hashedPassword[:16])
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(hashedPassword[16:])
	if err != nil {
		return false
	}

	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return bytes.Equal(hash, expectedHash)
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a user with the provided data
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.User true "User data"
// @Success 201 {object} map[string]interface{} "User registered successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 500 {object} map[string]interface{} "Failed to create user"
// @Router /users [post]
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

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to hash password",
			"error":   err.Error(),
		})
		return
	}
	user.Password = hashedPassword

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

// GetUserByID godoc
// @Summary Get a user by ID
// @Description Retrieve user information by user ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/{id} [get]
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

// UpdateUser godoc
// @Summary Update a user
// @Description Update user information
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body models.User true "User data"
// @Success 200 {object} map[string]interface{} "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 500 {object} map[string]interface{} "Failed to update user"
// @Router /users/{id} [put]
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

// DeleteUser godoc
// @Summary Delete a user
// @Description Delete user by ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 500 {object} map[string]interface{} "Failed to delete user"
// @Router /users/{id} [delete]
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

// LoginUser godoc
// @Summary Login a user
// @Description Authenticate user credentials
// @Tags users
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Email and Password"
// @Success 200 {object} map[string]interface{} "User logged in successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/login [post]
func (uc *UserController) LoginUser(c *gin.Context) {
	var loginRequest LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	user, err := uc.repo.GetUserByEmail(loginRequest.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user associated with this email",
		})
		return
	}

	// Use simple SHA256
	if !verifyPassword(user.Password, loginRequest.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "Invalid email or password",
		})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})
	jwtSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not generate token",
			"error":   err.Error(),
		})
		return
	}
	// Authentication is successful
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User logged in successfully",
		"data":    tokenString,
	})
}

// ForgotPassword godoc
// @Summary Request password reset code
// @Description Send a verification code to user's email for password reset
// @Tags users
// @Accept json
// @Produce json
// @Param forgotPassword body ForgotPasswordRequest true "User Email"
// @Success 200 {object} map[string]interface{} "Code sent successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data or email does not exist"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/forgot-password [post]
func (uc *UserController) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	if _, err := uc.repo.GetUserByEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Email's does not exist",
			"error":   err.Error(),
		})
		return
	}

	mailConfig := utils.LoadMailConfig()
	code := utils.GenerateVerificationCode()

	// Delete if any code from the previous one still exist
	uc.rp_repo.DeleteByEmail(req.Email)

	forget_password := &models.ResetPassword{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := uc.rp_repo.CreateResetPassword(forget_password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create forget password code",
			"error":   "Database error",
		})
		return
	}

	go func() {
		if err := utils.SendEmail(mailConfig, req.Email, "Reset Password", "Use this code to reset your password :  "+code); err != nil {
			log.Printf("Failed to send email to %s: %v", req.Email, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Code sent successfully",
		"data":    nil,
	})
}

// ResetPassword godoc
// @Summary Reset user password
// @Description Reset user password using verification code
// @Tags users
// @Accept json
// @Produce json
// @Param resetPassword body ResetPasswordRequest true "Email, Code, and New Password"
// @Success 200 {object} map[string]interface{} "Password has been reset successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data, code expired, or invalid password"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/reset-password [post]
func (uc *UserController) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	resetRecord, err := uc.rp_repo.FindByEmailAndCode(req.Email, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid or expired code",
			"error":   "Code not found",
		})
		return
	}

	if time.Now().After(resetRecord.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Code has expired",
			"error":   "Expired code",
		})
		return
	}

	user, err := uc.repo.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "User does not exist",
		})
		return
	}

	// Validate password
	if len(req.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Password must be at least 8 characters",
			"error":   "Invalid password",
		})
		return
	}

	// Hash the new password with simple SHA256
	hashedPassword, err := hashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to hash password",
			"error":   "Internal server error",
		})
		return
	}

	user.Password = hashedPassword
	if err := uc.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update password",
			"error":   "Database error",
		})
		return
	}

	// Delete reset password code
	if err := uc.rp_repo.DeleteByEmail(req.Email); err != nil {
		log.Printf("Failed to delete reset password code for %s: %v", req.Email, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password has been reset successfully",
		"data":    nil,
	})
}

// PatchUser godoc
// @Summary Patch current user
// @Description Update specific fields of the authenticated user's information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userData body map[string]interface{} true "User data to update"
// @Success 200 {object} map[string]interface{} "User patched successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Failed to update user"
// @Router /users/me [patch]
func (uc *UserController) PatchUser(c *gin.Context) {
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

	// Check if the user exists
	existingUser, err := uc.repo.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with the provided ID",
		})
		return
	}

	// Parse the patch data
	var patchData map[string]interface{}
	if err := c.ShouldBindJSON(&patchData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Handle password update specially if it's included
	if password, ok := patchData["password"].(string); ok {
		if len(password) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Password must be at least 8 characters",
				"error":   "Invalid password",
			})
			return
		}

		// Hash the new password with simple SHA256
		hashedPassword, err := hashPassword(password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to hash password",
				"error":   "Internal server error",
			})
			return
		}
		patchData["password"] = hashedPassword
	}

	// Prevent user from changing their role
	if _, hasRole := patchData["role"]; hasRole {
		// Check if user is admin
		userRole, roleExists := c.Get("role")
		if !roleExists || userRole.(string) != "admin" {
			// Non-admin users cannot change their role
			delete(patchData, "role")
		}
	}

	// Prevent changing email to one that already exists
	if email, hasEmail := patchData["email"].(string); hasEmail && email != existingUser.Email {
		// Check if email already exists
		_, err := uc.repo.GetUserByEmail(email)
		if err == nil { // No error means email exists
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Email already in use",
				"error":   "Email address is already registered",
			})
			return
		}
	}

	// Apply the patch to the user
	if err := uc.repo.PatchUser(userID.(uint), patchData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update user",
			"error":   err.Error(),
		})
		return
	}

	// Get the updated user
	updatedUser, err := uc.repo.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "User updated but failed to retrieve updated data",
			"error":   err.Error(),
		})
		return
	}

	// Hide sensitive information
	updatedUser.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User patched successfully",
		"data":    updatedUser,
	})
}

// GetCurrentUser godoc
// @Summary Get current user information
// @Description Retrieve information about the currently authenticated user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User information retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/me [get]
func (uc *UserController) GetCurrentUser(c *gin.Context) {
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

	// Retrieve the user
	user, err := uc.repo.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with this ID",
		})
		return
	}

	// Remove sensitive information
	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User information retrieved successfully",
		"data":    user,
	})
}
