package controllers

import (
	"context"
	"diabetify/internal/ml"
	"diabetify/internal/models"
	"diabetify/internal/openai"
	"diabetify/internal/repository"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PredictionController struct {
	repo         repository.PredictionRepository
	userRepo     *repository.UserRepository
	profileRepo  repository.UserProfileRepository
	activityRepo repository.ActivityRepository
	mlClient     ml.MLClient
}

func NewPredictionController(
	repo repository.PredictionRepository,
	userRepo *repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	activityRepo repository.ActivityRepository,
	mlClient ml.MLClient,
) *PredictionController {
	return &PredictionController{
		repo:         repo,
		userRepo:     userRepo,
		profileRepo:  profileRepo,
		activityRepo: activityRepo,
		mlClient:     mlClient,
	}
}

// MakePrediction godoc
// @Summary Make a prediction using user's profile data automatically
// @Description Automatically fetch user data from database and make diabetes risk prediction via gRPC (requires authentication)
// @Tags prediction
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Prediction result"
// @Failure 400 {object} map[string]interface{} "Incomplete user profile"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User profile not found"
// @Failure 500 {object} map[string]interface{} "Prediction failed"
// @Router /prediction [post]
func (pc *PredictionController) MakePrediction(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	// Get user data
	user, err := pc.userRepo.GetUserByID(userID.(uint))
	if err != nil {
		log.Printf("Error getting user by ID %d: %v", userID.(uint), err)
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   err.Error(),
		})
		return
	}

	// Get user profile
	profile, err := pc.profileRepo.FindByUserID(userID.(uint))
	log.Printf("Fetching profile for user ID %d", userID.(uint))
	if err != nil {
		log.Printf("Error getting user profile for user ID %d: %v", userID.(uint), err)
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User profile not found. Please complete your profile first.",
			"error":   err.Error(),
		})
		return
	}

	// Calculate features from user data
	features, featureInfo, err := pc.calculateFeaturesFromProfile(user, profile, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Incomplete profile data for prediction",
			"error":   err.Error(),
			"help":    "Please ensure all required profile fields are filled: age, weight, height, smoking status, macrosomic baby history, hypertension status",
		})
		return
	}

	// Create context with timeout for gRPC call
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Log the features being sent for prediction
	log.Printf("Features for prediction: %v", features)

	// Call the ML service via gRPC
	response, err := pc.mlClient.Predict(ctx, features)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Prediction failed",
			"error":   err.Error(),
		})
		return
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

		Age:             featureInfo["age"].(int),
		AgeContribution: ageContribution,
		AgeImpact:       ageImpact,

		BMI:             featureInfo["bmi"].(float64),
		BMIContribution: bmiContribution,
		BMIImpact:       bmiImpact,

		BrinkmanScore:             featureInfo["brinkman_score"].(float64),
		BrinkmanScoreContribution: brinkmanContribution,
		BrinkmanScoreImpact:       brinkmanImpact,

		IsHypertension:             featureInfo["is_hypertension"].(bool),
		IsHypertensionContribution: hypertensionContribution,
		IsHypertensionImpact:       hypertensionImpact,

		IsMacrosomicBaby:             featureInfo["is_macrosomic_baby"].(bool),
		IsMacrosomicBabyContribution: macrosomicContribution,
		IsMacrosomicBabyImpact:       macrosomicImpact,

		SmokingStatus:             fmt.Sprintf("%v", featureInfo["smoking_status"]), // Convert to string
		SmokingStatusContribution: smokingContribution,
		SmokingStatusImpact:       smokingImpact,

		PhysicalActivityMinutes:             featureInfo["physical_activity_minutes"].(int),
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

	// Update the last prediction time for the user
	now := time.Now()
	if err := pc.userRepo.UpdateLastPredictionTime(userID.(uint), &now); err != nil {
		log.Printf("Error updating last prediction time for user ID %d: %v", userID.(uint), err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update last prediction time",
			"error":   err.Error(),
		})
		return
	}
	// Return comprehensive response
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction successful via gRPC using your profile data",
		"data": gin.H{
			"prediction_id":   prediction.ID,
			"risk_score":      response.Prediction,
			"risk_percentage": response.Prediction * 100,
			"ml_service_time": response.ElapsedTime,
			"timestamp":       response.Timestamp,
			"user_data_used": gin.H{
				"age":                       featureInfo["age"],
				"smoking_status":            featureInfo["smoking_status"],
				"is_macrosomic_baby":        featureInfo["is_macrosomic_baby"],
				"brinkman_score":            featureInfo["brinkman_score"],
				"bmi":                       featureInfo["bmi"],
				"is_hypertension":           featureInfo["is_hypertension"],
				"physical_activity_minutes": featureInfo["physical_activity_minutes"],
			},
			"feature_explanations": response.Explanation,
		},
	})
}

func (pc *PredictionController) calculateFeaturesFromProfile(user *models.User, profile *models.UserProfile, userID uint) ([]float64, map[string]interface{}, error) {
	// Check BMI
	if profile.BMI == nil {
		return nil, nil, fmt.Errorf("BMI is required but not found")
	}
	bmi := *profile.BMI

	// Check MacrosomicBaby (safely handle nil)
	if profile.MacrosomicBaby == nil {
		return nil, nil, fmt.Errorf("macrosomic baby history is required but not found")
	}
	isMacrosomicBaby := *profile.MacrosomicBaby

	// Check Hypertension (safely handle nil)
	if profile.Hypertension == nil {
		return nil, nil, fmt.Errorf("hypertension status is required but not found")
	}
	isHypertension := *profile.Hypertension

	// Calculate age from string DOB
	if user.DOB == nil {
		return nil, nil, fmt.Errorf("date of birth is required but not found")
	}

	// Parse the DOB string - try multiple formats
	var dobTime time.Time
	var err error

	dobTime, err = time.Parse(time.RFC3339, *user.DOB)
	if err != nil {
		dobTime, err = time.Parse("2006-01-02", *user.DOB)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid date of birth format. Expected YYYY-MM-DD, got: %s", *user.DOB)
		}
	}

	// Calculate age
	now := time.Now()
	age := now.Year() - dobTime.Year()
	if now.YearDay() < dobTime.YearDay() {
		age--
	}

	// Calculate smoking status based on activity data (last 8 weeks)
	smokingStatus, err := pc.calculateSmokingStatus(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate smoking status: %v", err)
	}

	// Calculate Brinkman index from smoking activities
	brinkmanIndex, err := pc.calculateBrinkmanIndex(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate Brinkman index: %v", err)
	}

	// Calculate physical activity (average minutes per 8 weeks)
	physicalActivityMinutes, err := pc.calculatePhysicalActivityMinutes(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate physical activity: %v", err)
	}

	// Create features array for ML model
	features := []float64{
		float64(age),                     // 1. Age
		float64(smokingStatus),           // 2. Smoking status (0, 1, or 2)
		boolToFloat(isMacrosomicBaby),    // 3. Is macrosomic baby (0 or 1)
		brinkmanIndex,                    // 4. Brinkman index
		bmi,                              // 5. BMI
		boolToFloat(isHypertension),      // 6. Is hypertension (0 or 1)
		float64(physicalActivityMinutes), // 7. Physical activity minutes
	}

	featureInfo := map[string]interface{}{
		"age":                       age,
		"smoking_status":            smokingStatus,
		"is_macrosomic_baby":        isMacrosomicBaby,
		"brinkman_score":            brinkmanIndex,
		"bmi":                       bmi,
		"is_hypertension":           isHypertension,
		"physical_activity_minutes": physicalActivityMinutes,
	}

	return features, featureInfo, nil
}

// Helper functions
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// calculateSmokingStatus determines smoking status based on activity data (8 weeks)
// Returns: 0 = never smoked, 1 = used to smoke (>8 weeks ago), 2 = current smoker (within 8 weeks)
func (pc *PredictionController) calculateSmokingStatus(userID uint) (int, error) {
	// Get smoking activities from last 8 weeks
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -56)

	recentActivities, err := pc.activityRepo.GetActivitiesByUserIDAndTypeAndDateRange(userID, "smoke", startDate, endDate)
	if err != nil {
		return 0, err
	}

	// If user has smoking activities in last 8 weeks = current smoker
	if len(recentActivities) > 0 {
		return 2, nil
	}

	// Check if user has any smoking activities before 8 weeks ago
	historicalStartDate := endDate.AddDate(-10, 0, 0) // Check last 10 years
	allActivities, err := pc.activityRepo.GetActivitiesByUserIDAndTypeAndDateRange(userID, "smoke", historicalStartDate, startDate)
	if err != nil {
		return 0, err
	}

	// If user has smoking activities but not in recent 8 weeks = pernah merokok
	if len(allActivities) > 0 {
		return 1, nil
	}

	// No smoking activities found = never smoked
	return 0, nil
}

// calculateBrinkmanIndex calculates Brinkman index from smoking activities
func (pc *PredictionController) calculateBrinkmanIndex(userID uint) (float64, error) {
	// Get smoking activities from last 14 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -14)

	activities, err := pc.activityRepo.GetActivitiesByUserIDAndTypeAndDateRange(userID, "smoke", startDate, endDate)
	if err != nil {
		return 0.0, err
	}

	if len(activities) == 0 {
		return 0.0, nil
	}

	// Calculate average cigarettes per day
	totalCigarettes := 0
	for _, activity := range activities {
		totalCigarettes += activity.Value
	}

	avgCigarettesPerDay := float64(totalCigarettes) / 14.0 // Average over 14 days

	// Get estimated years of smoking from user profile
	profile, err := pc.profileRepo.FindByUserID(userID)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get user profile: %v", err)
	}
	estimatedYears := 0
	if profile.YearOfSmoking != nil {
		estimatedYears = *profile.YearOfSmoking
	} else {
		return 0.0, fmt.Errorf("year of smoking is required but not found in profile")
	}

	// Brinkman Index = cigarettes per day × years of smoking
	brinkmanIndex := avgCigarettesPerDay * float64(estimatedYears)

	return math.Round(brinkmanIndex*10) / 10, nil
}

// calculatePhysicalActivityMinutes calculates average workout minutes per 2 weeks
func (pc *PredictionController) calculatePhysicalActivityMinutes(userID uint) (int, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -14)

	activities, err := pc.activityRepo.GetActivitiesByUserIDAndTypeAndDateRange(userID, "workout", startDate, endDate)
	if err != nil {
		return 0, err
	}

	totalMinutes := 0
	for _, activity := range activities {
		totalMinutes += activity.Value
	}

	// Calculate average minutes per day over the 14 days
	return totalMinutes, nil
}

// TestMLConnection godoc
// @Summary Test ML service connection via gRPC
// @Description Test the gRPC connection to the ML service
// @Tags prediction
// @Produce json
// @Success 200 {object} map[string]interface{} "ML service is healthy"
// @Failure 500 {object} map[string]interface{} "ML service is not reachable"
// @Router /prediction/predict/health [get]
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

// GetUserPredictions godoc
// @Summary Get user's prediction history
// @Description Retrieve prediction history for the authenticated user
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Prediction history retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve prediction history"
// @Router /prediction/me [get]
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

	// Get Limit Params
	limitStr := c.Query("limit")
	limit := 10 // Default limit
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid limit parameter",
				"error":   "Limit must be a positive integer",
			})
			return
		}
	}

	predictions, err := pc.repo.GetPredictionsByUserID(userID.(uint), limit)
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

// GetPredictionsByDateRange godoc
// @Summary Get user's prediction history by date range
// @Description Retrieve prediction history for the authenticated user within a date range
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{} "Prediction history retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid date format"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve prediction history"
// @Router /prediction/me/date-range [get]
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

// GetPredictionByID godoc
// @Summary Get prediction by ID
// @Description Retrieve a specific prediction by ID
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Prediction ID"
// @Success 200 {object} map[string]interface{} "Prediction retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid prediction ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Prediction not found"
// @Router /prediction/{id} [get]
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

// DeletePrediction godoc
// @Summary Delete a prediction
// @Description Delete a specific prediction by ID
// @Tags prediction
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Prediction ID"
// @Success 200 {object} map[string]interface{} "Prediction deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid prediction ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Prediction not found"
// @Router /prediction/{id} [delete]
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

func (pc *PredictionController) GetPredictionScoreByDate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
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

	scores, err := pc.repo.GetPredictionScoreByUserIDAndDateRange(userID.(uint), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve prediction score",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Prediction score retrieved successfully",
		"data":    scores,
	})
}

// GetLatestPredictionExplanation godoc
// @Summary Get latest prediction with LLM explanation for current user
// @Description Get the most recent prediction with detailed LLM explanation for the authenticated user
// @Tags prediction
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.Prediction "Latest prediction with explanation"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "No prediction found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /prediction/me/latest [get]
func (pc *PredictionController) GetLatestPredictionExplanation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	// Get latest prediction
	prediction, err := pc.repo.GetLatestPredictionByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "No prediction found",
			"error":   err.Error(),
		})
		return
	}

	// If the prediction doesn't have an LLM explanation, generate one
	// if prediction.LLMExplanation == "" {
		openaiClient, err := openai.NewClient()
		if err != nil {
			log.Printf("Error creating OpenAI client: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to initialize OpenAI client",
				"error":   err.Error(),
			})
			return
		}

		factors := map[string]struct {
			Contribution float64
			Impact       float64
		}{
			"Age":                     {Contribution: prediction.AgeContribution, Impact: prediction.AgeImpact},
			"BMI":                     {Contribution: prediction.BMIContribution, Impact: prediction.BMIImpact},
			"Brinkman Score":          {Contribution: prediction.BrinkmanScoreContribution, Impact: prediction.BrinkmanScoreImpact},
			"Hypertension":            {Contribution: prediction.IsHypertensionContribution, Impact: prediction.IsHypertensionImpact},
			"Macrosomic Baby":         {Contribution: prediction.IsMacrosomicBabyContribution, Impact: prediction.IsMacrosomicBabyImpact},
			"Smoking Status":          {Contribution: prediction.SmokingStatusContribution, Impact: prediction.SmokingStatusImpact},
			"Physical Activity Level": {Contribution: prediction.PhysicalActivityMinutesContribution, Impact: prediction.PhysicalActivityMinutesImpact},
		}

		llmExplanation, err := openaiClient.GeneratePredictionExplanation(prediction.RiskScore, factors)
		if err != nil {
			log.Printf("Error generating LLM explanation: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to generate LLM explanation",
				"error":   err.Error(),
			})
			return
		}

		// Update prediction with LLM explanation
		// prediction.LLMExplanation = llmExplanation
		// if err := pc.repo.UpdatePrediction(prediction); err != nil {
		// 	log.Printf("Error updating prediction with LLM explanation: %v", err)
		// 	c.JSON(http.StatusInternalServerError, gin.H{
		// 		"status":  "error",
		// 		"message": "Failed to update prediction with LLM explanation",
		// 		"error":   err.Error(),
		// 	})
		// 	return
		// }
	// }

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Latest prediction retrieved successfully",
		"data": llmExplanation,
	})
}
