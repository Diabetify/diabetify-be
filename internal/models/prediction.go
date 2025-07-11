package models

import (
	"time"

	"gorm.io/gorm"
)

type Prediction struct {
	ID                                    uint           `gorm:"primaryKey" json:"id" example:"1"`
	CreatedAt                             time.Time      `gorm:"index" json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt                             time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt                             gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	UserID                                uint           `gorm:"index" json:"user_id" example:"1"`
	User                                  User           `gorm:"foreignKey:UserID" json:"-"`
	RiskScore                             float64        `json:"risk_score" example:"0.75"`
	Age                                   int            `json:"age" example:"30"`
	AgeShap                               float64        `json:"age_shap" example:"0.05"`
	AgeContribution                       float64        `json:"age_contribution" example:"0.1"`
	AgeImpact                             float64        `json:"age_impact" example:"0.2"`
	AgeExplanation                        string         `gorm:"type:text" json:"age_explanation"`
	BMI                                   float64        `json:"bmi" example:"22.5"`
	BMIShap                               float64        `json:"bmi_shap" example:"0.05"`
	BMIContribution                       float64        `json:"bmi_contribution" example:"0.15"`
	BMIImpact                             float64        `json:"bmi_impact" example:"0.25"`
	BMIExplanation                        string         `gorm:"type:text" json:"bmi_explanation"`
	AvgSmokeCount                         int            `json:"avg_smoke_count" example:"14"`
	BrinkmanScore                         int            `gorm:"column:brinkman_score;check:brinkman_score IN (0,1,2,3)" json:"brinkman_score" example:"0" validate:"min=0,max=3"`
	BrinkmanScoreShap                     float64        `json:"brinkman_score_shap" example:"0.05"`
	BrinkmanScoreContribution             float64        `json:"brinkman_score_contribution" example:"0.2"`
	BrinkmanScoreImpact                   float64        `json:"brinkman_score_impact" example:"0.3"`
	BrinkmanScoreExplanation              string         `gorm:"type:text" json:"brinkman_score_explanation"`
	IsHypertension                        bool           `json:"is_hypertension" example:"true"`
	IsHypertensionShap                    float64        `json:"is_hypertension_shap" example:"0.05"`
	IsHypertensionContribution            float64        `json:"is_hypertension_contribution" example:"0.1"`
	IsHypertensionImpact                  float64        `json:"is_hypertension_impact" example:"0.2"`
	IsHypertensionExplanation             string         `gorm:"type:text" json:"is_hypertension_explanation"`
	IsCholesterol                         bool           `json:"is_cholesterol" example:"true"`
	IsCholesterolShap                     float64        `json:"is_cholesterol_shap" example:"0.05"`
	IsCholesterolContribution             float64        `json:"is_cholesterol_contribution" example:"0.1"`
	IsCholesterolImpact                   float64        `json:"is_cholesterol_impact" example:"0.2"`
	IsCholesterolExplanation              string         `gorm:"type:text" json:"is_cholesterol_explanation"`
	IsBloodline                           bool           `json:"is_bloodline" example:"true"`
	IsBloodlineShap                       float64        `json:"is_bloodline_shap" example:"0.05"`
	IsBloodlineContribution               float64        `json:"is_bloodline_contribution" example:"0.1"`
	IsBloodlineImpact                     float64        `json:"is_bloodline_impact" example:"0.2"`
	IsBloodlineExplanation                string         `gorm:"type:text" json:"is_bloodline_explanation"`
	IsMacrosomicBaby                      int            `gorm:"column:macrosomic_baby" json:"macrosomic_baby" example:"0"`
	IsMacrosomicBabyShap                  float64        `json:"is_macrosomic_baby_shap" example:"0.05"`
	IsMacrosomicBabyContribution          float64        `json:"is_macrosomic_baby_contribution" example:"0.05"`
	IsMacrosomicBabyImpact                float64        `json:"is_macrosomic_baby_impact" example:"0.1"`
	IsMacrosomicBabyExplanation           string         `gorm:"type:text" json:"is_macrosomic_baby_explanation"`
	SmokingStatus                         int            `gorm:"column:smoking_status;check:smoking_status IN (0,1,2)" json:"smoking_status" example:"0" validate:"min=0,max=2"`
	SmokingStatusShap                     float64        `json:"smoking_status_shap" example:"0.05"`
	SmokingStatusContribution             float64        `json:"smoking_status_contribution" example:"0.1"`
	SmokingStatusImpact                   float64        `json:"smoking_status_impact" example:"0.2"`
	SmokingStatusExplanation              string         `gorm:"type:text" json:"smoking_status_explanation"`
	PhysicalActivityFrequency             int            `json:"physical_activity_frequency" example:"150"`
	PhysicalActivityFrequencyShap         float64        `json:"physical_activity_frequency_shap" example:"0.05"`
	PhysicalActivityFrequencyContribution float64        `json:"physical_activity_frequency_contribution" example:"0.1"`
	PhysicalActivityFrequencyImpact       float64        `json:"physical_activity_frequency_impact" example:"0.2"`
	PhysicalActivityFrequencyExplanation  string         `gorm:"type:text" json:"physical_activity_frequency_explanation"`
	PredictionSummary                     string         `gorm:"type:text" json:"prediction_summary" example:"This user has a moderate risk of diabetes."`
}

func (p *Prediction) GetShardKey() int {
	return int(p.UserID)
}

func (p *Prediction) TableName() string {
	return "predictions"
}

type PredictionRequest struct {
	Features []float64 `json:"features" binding:"required"`
}

type ExplanationItem struct {
	Shap         float64 `json:"shap"`
	Contribution float64 `json:"contribution"`
	Impact       int     `json:"impact"`
}

type PredictionResponse struct {
	Prediction  float64                    `json:"prediction"`
	Explanation map[string]ExplanationItem `json:"explanation,omitempty"`
	ElapsedTime float64                    `json:"elapsed_time_seconds,omitempty"`
	Timestamp   time.Time                  `json:"timestamp"`
}

type UpdateModelRequest struct {
	XNew   [][]float64 `json:"x_new" binding:"required"`
	YNew   []float64   `json:"y_new" binding:"required"`
	XVal   [][]float64 `json:"x_val" binding:"required"`
	YVal   []float64   `json:"y_val" binding:"required"`
	Epochs int         `json:"epochs" binding:"required,min=1"`
}

type UpdateModelResponse struct {
	Status      string    `json:"status"`
	AUCBefore   float64   `json:"auc_before"`
	AUCAfter    float64   `json:"auc_after"`
	PRAUCBefore float64   `json:"pr_auc_before"`
	PRAUCAfter  float64   `json:"pr_auc_after"`
	ElapsedTime float64   `json:"elapsed_time"`
	Timestamp   time.Time `json:"timestamp"`
}

type JobProgressUpdate struct {
	JobID    string `json:"job_id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Step     string `json:"step"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

type AsyncPredictionResponse struct {
	JobID          string      `json:"job_id"`
	Status         string      `json:"status"`
	Progress       int         `json:"progress"`
	Step           string      `json:"step"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty"`
	ProcessingTime *float64    `json:"processing_time,omitempty"` // in seconds
	Prediction     *Prediction `json:"prediction,omitempty"`
	Error          string      `json:"error,omitempty"`
}
