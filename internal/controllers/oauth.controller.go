package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type OauthController struct {
	userRepo *repository.UserRepository
}

func NewOauthController(userRepo *repository.UserRepository) *OauthController {
	return &OauthController{userRepo: userRepo}
}
func (oc *OauthController) GoogleAuth(c *gin.Context) {
	var authRequest struct {
		Token string `json:"token"`
	}

	if err := c.ShouldBindJSON(&authRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + authRequest.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to verify token with Google",
			"error":   err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid Google ID token",
			"error":   "Token verification failed",
		})
		return
	}

	var tokenInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to decode token info",
			"error":   err.Error(),
		})
		return
	}

	email, ok := tokenInfo["email"].(string)
	if !ok || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Email not found in token",
		})
		return
	}

	name, _ := tokenInfo["name"].(string)
	user, err := oc.userRepo.GetUserByEmail(email)
	isNewUser := err != nil

	if isNewUser {
		// Create new user (sign-up)
		newUser := models.User{
			Email:    email,
			Name:     name,
			Password: "",
		}

		err = oc.userRepo.CreateUser(&newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to create user account",
				"error":   err.Error(),
			})
			return
		}
	}

	// Generate JWT
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	jwtSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
	tokenString, err := jwtToken.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not generate token",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Google authentication successful",
		"data":    tokenString,
	})
}
