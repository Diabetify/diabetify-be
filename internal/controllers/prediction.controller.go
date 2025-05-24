package controllers

import (
	"context"
	"diabetify/internal/ml"
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PredictionController handles prediction-related requests
type PredictionController struct {
	repo     repository.PredictionRepository
	mlClient ml.MLClient
}

// NewPredictionController creates a new prediction controller
func NewPredictionController(
	repo repository.PredictionRepository,
	mlClient ml.MLClient,
) *PredictionController {
	return &PredictionController{
		repo:     repo,
		mlClient: mlClient,
	}
}

// MakePrediction godoc
// @Summary Make a prediction using ML model via gRPC
// @Description Send features to ML model via gRPC and get a prediction (requires authentication)
// @Tags prediction
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param prediction body models.PredictionRequest true "Features array: [age, smoking_status, is_macrosomic_baby, brinkman_index, BMI, is_hypertension, physical_activity_minutes]"
// @Success 200 {object} map[string]interface{} "Prediction result"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Prediction failed"
// @Router /predict [post]
func (pc *PredictionController) MakePrediction(c *gin.Context) {
	var request models.PredictionRequest

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Validate features array length
	if len(request.Features) != 7 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":   "error",
			"message":  "Invalid features array: expected 7 features",
			"expected": "Features order: [age, smoking_status, is_macrosomic_baby, brinkman_index, BMI, is_hypertension, physical_activity_minutes]",
		})
		return
	}

	// Create context with timeout for gRPC call
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Call the ML service via gRPC
	response, err := pc.mlClient.Predict(ctx, request.Features)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Prediction failed",
			"error":   err.Error(),
		})
		return
	}

	// Convert the numeric features to proper types
	age := int(request.Features[0])
	smokingStatusValue := int(request.Features[1])
	isMacrosomicBaby := request.Features[2] > 0
	brinkmanScore := request.Features[3]
	bmi := request.Features[4]
	isHypertension := request.Features[5] > 0
	physicalActivityMinutes := int(request.Features[6])

	// Convert smoking status numeric value to string
	var smokingStatus string
	switch smokingStatusValue {
	case 0:
		smokingStatus = "non_smoker"
	case 1:
		smokingStatus = "smoker"
	default:
		smokingStatus = "unknown"
	}

	// Helper function to safely extract explanation values
	getExplanation := func(key string) (float64, float64) {
		if exp, exists := response.Explanation[key]; exists {
			return exp.Contribution, float64(exp.Impact)
		}
		return 0.0, 0.0
	}

	// Extract explanations safely
	ageContribution, ageImpact := getExplanation("age")
	bmiContribution, bmiImpact := getExplanation("BMI")
	brinkmanContribution, brinkmanImpact := getExplanation("brinkman_index")
	hypertensionContribution, hypertensionImpact := getExplanation("is_hypertension")
	macrosomicContribution, macrosomicImpact := getExplanation("is_macrosomic_baby")
	smokingContribution, smokingImpact := getExplanation("smoking_status")
	activityContribution, activityImpact := getExplanation("physical_activity_minute")

	// Create a new prediction record for database
	prediction := &models.Prediction{
		UserID:    userID.(uint),
		RiskScore: response.Prediction,

		Age:             age,
		AgeContribution: ageContribution,
		AgeImpact:       ageImpact,

		BMI:             bmi,
		BMIContribution: bmiContribution,
		BMIImpact:       bmiImpact,

		BrinkmanScore:             brinkmanScore,
		BrinkmanScoreContribution: brinkmanContribution,
		BrinkmanScoreImpact:       brinkmanImpact,

		IsHypertension:             isHypertension,
		IsHypertensionContribution: hypertensionContribution,
		IsHypertensionImpact:       hypertensionImpact,

		IsMacrosomicBaby:             isMacrosomicBaby,
		IsMacrosomicBabyContribution: macrosomicContribution,
		IsMacrosomicBabyImpact:       macrosomicImpact,

		SmokingStatus:             smokingStatus,
		SmokingStatusContribution: smokingContribution,
		SmokingStatusImpact:       smokingImpact,

		PhysicalActivityMinutes:             physicalActivityMinutes,
		PhysicalActivityMinutesContribution: activityContribution,
		PhysicalActivityMinutesImpact:       activityImpact,
	}

	// Save to database
	if err := pc.repo.SavePrediction(prediction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to save prediction",
			"error":   err.Error(),
		})
		return
	}

	// Return comprehensive response
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction successful via gRPC",
		"data": gin.H{
			"prediction_id":   prediction.ID,
			"risk_score":      response.Prediction,
			"risk_percentage": response.Prediction * 100,
			"ml_service_time": response.ElapsedTime,
			"timestamp":       response.Timestamp,
			"features_analyzed": gin.H{
				"age":                       age,
				"smoking_status":            smokingStatus,
				"is_macrosomic_baby":        isMacrosomicBaby,
				"brinkman_score":            brinkmanScore,
				"bmi":                       bmi,
				"is_hypertension":           isHypertension,
				"physical_activity_minutes": physicalActivityMinutes,
			},
			"feature_explanations": response.Explanation,
		},
	})
}

// TestMLConnection godoc
// @Summary Test ML service connection via gRPC
// @Description Test the gRPC connection to the ML service
// @Tags prediction
// @Produce json
// @Success 200 {object} map[string]interface{} "ML service is healthy"
// @Failure 500 {object} map[string]interface{} "ML service is not reachable"
// @Router /predict/health [get]
func (pc *PredictionController) TestMLConnection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := pc.mlClient.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "ML service is not reachable via gRPC",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"message":   "ML service is healthy via gRPC",
		"timestamp": time.Now(),
	})
}

// GetUserPredictions - Keep your existing implementation
func (pc *PredictionController) GetUserPredictions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	predictions, err := pc.repo.GetPredictionsByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve prediction history",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction history retrieved successfully",
		"data":    predictions,
	})
}

// GetPredictionsByDateRange - Keep your existing implementation
func (pc *PredictionController) GetPredictionsByDateRange(c *gin.Context) {
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

	endDate = endDate.Add(24 * time.Hour).Add(-time.Second)

	predictions, err := pc.repo.GetPredictionsByUserIDAndDateRange(userID.(uint), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve prediction history",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction history retrieved successfully",
		"data":    predictions,
	})
}

// GetPredictionByID - Keep your existing implementation
func (pc *PredictionController) GetPredictionByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid prediction ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	prediction, err := pc.repo.GetPredictionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Prediction not found",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if prediction.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Access denied: prediction belongs to a different user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction retrieved successfully",
		"data":    prediction,
	})
}

// DeletePrediction - Keep your existing implementation
func (pc *PredictionController) DeletePrediction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid prediction ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	prediction, err := pc.repo.GetPredictionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Prediction not found",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	if prediction.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Access denied: prediction belongs to a different user",
		})
		return
	}

	if err := pc.repo.DeletePrediction(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete prediction",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction deleted successfully",
	})
}
