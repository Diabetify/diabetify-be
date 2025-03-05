package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"diabetify/internal/utils"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type EmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type VerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

type VerificationController struct {
	verificationRepo *repository.VerificationRepository
	userRepo         *repository.UserRepository
	mailConfig       utils.MailConfig
}

func NewVerificationController(verificationRepo *repository.VerificationRepository, userRepo *repository.UserRepository) *VerificationController {
	mailConfig := utils.LoadMailConfig()
	return &VerificationController{
		verificationRepo: verificationRepo,
		userRepo:         userRepo,
		mailConfig:       mailConfig,
	}
}

// SendVerificationCode godoc
// @Summary Send a verification code to user's email
// @Description Sends a 6-digit verification code to the specified email address
// @Tags verification
// @Accept json
// @Produce json
// @Param email body EmailRequest true "User email"
// @Success 200 {object} map[string]interface{} "Verification code sent successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Failed to create verification code"
// @Router /verify/send [post]
func (vc *VerificationController) SendVerificationCode(c *gin.Context) {
	var req EmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Check if user exists
	_, err := vc.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No account associated with this email",
		})
		return
	}

	// Generate a 6-digit code
	code := utils.GenerateVerificationCode()

	// Store in DB (delete old if exists)
	vc.verificationRepo.DeleteByEmail(req.Email)

	verification := &models.Verification{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := vc.verificationRepo.CreateVerification(verification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create verification code",
			"error":   "Database error",
		})
		return
	}

	// Send email asynchronously
	go func() {
		if err := utils.SendEmail(vc.mailConfig, req.Email, "Verification Code", "Your verification code is: "+code); err != nil {
			log.Printf("Failed to send email to %s: %v", req.Email, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Verification code sent successfully",
		"data":    nil,
	})
}

// VerifyCode godoc
// @Summary Verify a user's verification code
// @Description Verifies the provided code for the user's email
// @Tags verification
// @Accept json
// @Produce json
// @Param verification body VerificationRequest true "Verification details"
// @Success 200 {object} map[string]interface{} "Verification successful"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Invalid or expired verification code"
// @Failure 500 {object} map[string]interface{} "Failed to verify user"
// @Router /verify [post]
func (vc *VerificationController) VerifyCode(c *gin.Context) {
	var req VerificationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	_, err := vc.verificationRepo.FindByEmailAndCode(req.Email, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid or expired verification code",
			"error":   "Code is incorrect or has expired",
		})
		return
	}

	if err := vc.userRepo.SetUserVerified(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to verify user",
			"error":   "Database error",
		})
		return
	}

	vc.verificationRepo.DeleteByEmail(req.Email)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Verification successful",
		"data":    nil,
	})
}

// ResendVerificationCode godoc
// @Summary Resend the verification code
// @Description Resends the verification code to the user's email
// @Tags verification
// @Accept json
// @Produce json
// @Param email body EmailRequest true "User email"
// @Success 200 {object} map[string]interface{} "Verification code resent successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Failed to create verification code"
// @Router /verify/resend [post]
func (vc *VerificationController) ResendVerificationCode(c *gin.Context) {
	vc.SendVerificationCode(c)
}
