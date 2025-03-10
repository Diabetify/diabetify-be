package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth/gothic"
)

type OauthController struct {
	userRepo *repository.UserRepository
}

func NewOauthController(userRepo *repository.UserRepository) *OauthController {
	return &OauthController{userRepo: userRepo}
}

func (oc *OauthController) GoogleAuth(c *gin.Context) {
	q := c.Request.URL.Query()
	q.Add("provider", "google")

	if callbackURL := c.Query("callback_url"); callbackURL != "" {
		q.Add("state", callbackURL)
	}

	c.Request.URL.RawQuery = q.Encode()
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (oc *OauthController) GoogleCallback(c *gin.Context) {
	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication failed",
			"error":   err.Error(),
		})
		return
	}

	dbUser, err := oc.userRepo.GetUserByEmail(user.Email)

	if err != nil {
		newUser := &models.User{
			Email:    user.Email,
			Name:     user.Name,
			Password: "",
			Verified: true,
		}

		if err := oc.userRepo.CreateUser(newUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to create user",
				"error":   err.Error(),
			})
			return
		}

		dbUser = newUser
	} else if !dbUser.Verified {
		dbUser.Verified = true
		if err := oc.userRepo.UpdateUser(dbUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to update user verification status",
				"error":   err.Error(),
			})
			return
		}
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": dbUser.ID,
		"email":   dbUser.Email,
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

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User authenticated successfully",
		"data": gin.H{
			"token": tokenString,
			"user": gin.H{
				"id":     dbUser.ID,
				"name":   user.Name,
				"email":  user.Email,
				"avatar": user.AvatarURL,
			},
		},
	})
}

func (oc *OauthController) Logout(c *gin.Context) {
	gothic.Logout(c.Writer, c.Request)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logged out successfully",
		"data":    nil,
	})
}
