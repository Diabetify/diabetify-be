package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ChatMessage struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type FactorExplanation struct {
	Factor              string  `json:"factor"`
	Value               string  `json:"value"`
	Impact              string  `json:"impact"`
	Shap                float64 `json:"shap"`
	Contribution        float64 `json:"contribution"`
	ContributionPercent string  `json:"contribution_percent"`
	Explanation         string  `json:"explanation"`
}

type PredictionExplanationResponse struct {
	Summary  string    `json:"summary"`
	Features []Feature `json:"features"`
}

type Feature struct {
	FeatureName string `json:"feature_name"`
	Explanation string `json:"explanation"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type FeatureInfo struct {
	Name                    string
	Alias                   string
	Description             string
	GlobalImportanceInsight string
	ImportanceRank          int
}

func getFeatureDefinitions() map[string]FeatureInfo {
	return map[string]FeatureInfo{
		"age": {
			Name:                    "age",
			Alias:                   "Usia",
			Description:             "User's age in years (e.g., 50).",
			GlobalImportanceInsight: "Most significant feature. Higher age strongly increases diabetes risk (SHAP up to +0.15), while lower age decreases it (SHAP down to -0.15).",
			ImportanceRank:          1,
		},
		"bmi": {
			Name:                    "bmi",
			Alias:                   "Indeks Massa Tubuh",
			Description:             "Body Mass Index (e.g., 20.5), based on Asian population classifications: <18.5=Underweight, 18.5-22.9=Normal, 23.0-24.9=Overweight, 25.0-29.9=Obese I, ≥30.0=Obese II.",
			GlobalImportanceInsight: "Second most influential feature. High BMI values strongly increase diabetes risk (SHAP up to +0.20), while low BMI values decrease it (SHAP down to -0.10).",
			ImportanceRank:          2,
		},
		"is_hypertension": {
			Name:                    "is_hypertension",
			Alias:                   "Hipertensi",
			Description:             "Indicates if the user has hypertension (high blood pressure): 0=No, 1=Yes.",
			GlobalImportanceInsight: "A 'Yes' diagnosis increases diabetes risk (SHAP ~+0.05), while 'No' decreases it (SHAP ~-0.05).",
			ImportanceRank:          3,
		},
		"smoking_status": {
			Name:                    "smoking_status",
			Alias:                   "Status Merokok",
			Description:             "User's smoking status: 0=Never smoked, 1=Former smoker, 2=Active smoker.",
			GlobalImportanceInsight: "Being a current or former smoker increases diabetes risk (SHAP up to +0.05). Never smoking slightly decreases the risk.",
			ImportanceRank:          4,
		},
		"is_macrosomic_baby": {
			Name:                    "is_macrosomic_baby",
			Alias:                   "Riwayat Melahirkan Bayi Besar",
			Description:             "History of giving birth to a baby over 4 kg: 0=No, 1=Yes, 2=Not applicable (never pregnant).",
			GlobalImportanceInsight: "A 'Yes' history increases diabetes risk (SHAP up to +0.03), while 'No' has a reducing effect (SHAP ~-0.025).",
			ImportanceRank:          5,
		},
		"brinkman_score": {
			Name:                    "brinkman_score",
			Alias:                   "Indeks Brinkman",
			Description:             "Measures lifetime tobacco exposure, represented as a categorized value: 0=Never smoked, 1=Mild smoker, 2=Moderate smoker, 3=Heavy smoker. This is a preprocessed category, not the raw Brinkman Index.",
			GlobalImportanceInsight: "Higher scores increase diabetes risk (SHAP up to +0.05), while lower scores decrease it (SHAP down to -0.05).",
			ImportanceRank:          6,
		},
		"is_cholesterol": {
			Name:                    "is_cholesterol",
			Alias:                   "Kolesterol Tinggi",
			Description:             "Indicates if the user has been diagnosed with high cholesterol: 0=No, 1=Yes.",
			GlobalImportanceInsight: "A 'Yes' diagnosis increases diabetes risk (SHAP up to +0.03). Normal cholesterol levels have a slight risk-reducing effect.",
			ImportanceRank:          7,
		},
		"is_bloodline": {
			Name:                    "is_bloodline",
			Alias:                   "Riwayat Keluarga dengan Diabetes",
			Description:             "Indicates if a parent died from diabetes: 0=No, 1=Yes.",
			GlobalImportanceInsight: "A 'Yes' history increases diabetes risk (SHAP up to +0.05). A 'No' history has a minimal risk-reducing impact.",
			ImportanceRank:          8,
		},
		"physical_activity_frequency": {
			Name:                    "physical_activity_frequency",
			Alias:                   "Frekuensi Aktivitas Fisik Sedang",
			Description:             "Days per week the user performs moderate-intensity physical activity.",
			GlobalImportanceInsight: "Least impactful predictor. More activity slightly decreases risk (SHAP ~-0.02), while less activity slightly increases it.",
			ImportanceRank:          9,
		},
	}
}

func getAliasToFeatureMapping() map[string]string {
	featureDefinitions := getFeatureDefinitions()
	aliasToFeature := make(map[string]string)

	for featureName, info := range featureDefinitions {
		aliasToFeature[info.Alias] = featureName
	}

	return aliasToFeature
}

func buildFeatureTable(features map[string]FeatureInfo, factorKeys []string) string {
	var table strings.Builder
	table.WriteString("| Feature Name | Feature Alias | Feature Description |\n")
	table.WriteString("|--------------|---------------|---------------------|\n")

	for _, factor := range factorKeys {
		if info, exists := features[factor]; exists {
			table.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				info.Name, info.Alias, info.Description))
		}
	}

	return table.String()
}

func buildGlobalImportanceExplanation(features map[string]FeatureInfo) string {
	var explanation strings.Builder
	explanation.WriteString("## Global Feature Importance Analysis\n\n")
	explanation.WriteString("Based on the analysis of the entire dataset, here are the key insights about how each feature typically influences diabetes risk:\n\n")

	importanceOrder := []string{
		"age", "bmi", "is_hypertension", "smoking_status", "is_macrosomic_baby",
		"brinkman_score", "is_cholesterol", "is_bloodline", "physical_activity_frequency",
	}

	for i, featureName := range importanceOrder {
		if info, exists := features[featureName]; exists {
			explanation.WriteString(fmt.Sprintf("%d. **%s (%s)**: %s\n\n",
				i+1, info.Alias, info.Name, info.GlobalImportanceInsight))
		}
	}

	return explanation.String()
}

func NewClient() (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}, nil
}

func (c *Client) GeneratePredictionExplanation(ctx context.Context, prediction float64, factors map[string]struct {
	Value        string
	Shap         float64
	Contribution float64
	Impact       float64
}) (map[string]FactorExplanation, string, TokenUsage, error) {

	featureDefinitions := getFeatureDefinitions()
	aliasToFeature := getAliasToFeatureMapping()

	factorKeys := make([]string, 0, len(factors))
	for factor := range factors {
		factorKeys = append(factorKeys, factor)
	}

	featureTable := buildFeatureTable(featureDefinitions, factorKeys)
	globalImportanceExplanation := buildGlobalImportanceExplanation(featureDefinitions)

	var shapTable strings.Builder
	shapTable.WriteString("| Feature Name | Input Value | SHAP Value | Contribution % |\n")
	shapTable.WriteString("|--------------|-------------|------------|----------------|\n")
	for factor, details := range factors {
		shapTable.WriteString(fmt.Sprintf("| %s | %s | %.6f | %.1f%% |\n",
			factor, details.Value, details.Shap, details.Contribution*100))
	}

	diabetesRiskPercentage := prediction * 100

	systemPrompt := fmt.Sprintf(`# Diabetes Prediction Explanation System

## 1. PERSONA & TONE
- **Role**: Medical AI Explainer for diabetes risk.
- **Audience**: Address the user as "Anda" (formal Indonesian).
- **Language**: Use simple, everyday Bahasa Indonesia. Avoid technical jargon unless explained.

---

## 2. CORE TASK
Generate a personalized, easy-to-understand explanation for a user's diabetes risk prediction based on their data. The output must be a valid JSON object.

---

## 3. KNOWLEDGE BASE & RULES

### A. Core Concepts
- **SHAP Value**: Represents the actual push a feature gives to the prediction.
  - Positive values **increase** risk.
  - Negative values **decrease** risk.
- **Contribution Percentage**: Represents the feature's **share of influence** relative to all other features. It is **NOT** the amount the risk increases by.

### B. SHAP Impact Levels
Use the **absolute SHAP value** to describe the strength of the impact:
- **|SHAP| > 0.2**: Very Strong ("sangat kuat" or "sangat signifikan")
- **|SHAP| 0.1 - 0.2**: Strong ("kuat" or "signifikan")
- **|SHAP| 0.05 - 0.1**: Moderate ("cukup" or "moderat")
- **|SHAP| < 0.05**: Slight ("sedikit" or "kecil")

### C. Feature Definitions
The following table defines the features used in the model. Use the "Feature Alias" in your explanations.

%s

### D. Global Feature Importance
The following list summarizes the global feature importance insights based on the entire dataset. Use these insights to explain how each feature typically influences diabetes risk.
%s

---

## 4. OUTPUT FORMAT
The output MUST be a **valid JSON object** with the following structure:
{
  "summary": "string",
  "features": [
    {
      "feature_name": "string",
      "explanation": "string"
    }
  ]
}

---

## 5. CONTENT REQUIREMENTS

### 'summary'
A 2-sentence summary
1. State the overall diabetes risk percentage and its category (Low: <35%%, Moderate: 35-55%%, High: 55-70%%, Very High: >70%%).
2. From only the factors that **increase risk (positive SHAP value)**, identify the top 1-3 with the highest contribution percentages.

### 'features'
An array of explanations for each feature
- **feature_name**: Use the original English feature name (e.g., "age", "bmi").  
- **explanation**: A 2-sentence explanation:
  - **Sentence 1**: State the user's value and its impact. **Crucially**, use the correct phrasing for contribution percentage. Mention the impact strength based on the SHAP value.
    - For categorical values (0, 1, 2), use the human-readable label (e.g., "pernah merokok" instead of "1").
    - **CORRECT PHRASING**: "...berkontribusi sebesar [Contribution %%] dari total pengaruh semua faktor."
    - **INCORRECT PHRASING**: "...menaikkan risiko Anda sebesar [Contribution %%]."
  - **Sentence 2**: Explain the general relationship between this feature and diabetes risk using the **Global Feature Importance** information.
  	Start by describing how the user's value (e.g., low/high/certain category) typically affects the risk,  
  	then contrast it with how the **opposite value or category** affects the risk.  

---

## 6. FEW-SHOT EXAMPLES

### Example 1: High BMI Impact (Strong Impact)
**Input Data**: BMI = 28.5, SHAP = +0.15, Contribution = 25.0%%
**Expected Output**:
"explanation": "Indeks massa tubuh Anda yang tergolong Obesitas I (28.5) memberikan pengaruh kuat yang menaikkan risiko Anda, berkontribusi sebesar 25.0%% dari total pengaruh semua faktor. Secara umum, indeks massa tubuh yang tinggi cenderung meningkatkan risiko diabetes, sedangkan indeks massa tubuh yang normal akan menurunkannya."

### Example 2: Young Age Factor (Moderate Impact)
**Input Data**: Age = 25, SHAP = -0.08, Contribution = 12.0%%
**Expected Output**:
"explanation": "Usia Anda yang tergolong muda (25 tahun) memberikan pengaruh moderat yang menurunkan risiko, berkontribusi sebesar 12.0%% dari total pengaruh semua faktor. Usia muda cenderung menurunkan risiko diabetes, sementara usia yang lebih tua cenderung meningkatkannya."

### Example 3: Summary
**Input Data**: Overall Risk = 65%%, Top factors: BMI (25.0%%), is_bloodline (18.0%%)
**Expected Output**:
"summary": "Berdasarkan analisis data, risiko diabetes Anda adalah 65.0%% yang tergolong tinggi. Faktor utama yang mendorong kenaikan risiko ini adalah indeks massa tubuh (25.0%%) dan riwayat keluarga (18.0%%)."

`, featureTable, globalImportanceExplanation)

	userPrompt := fmt.Sprintf(`Please analyze the user's diabetes prediction data below and generate the JSON explanation.

Overall Diabetes Risk: %.1f%%

Feature Analysis:
%s
`, diabetesRiskPercentage, shapTable.String())

	messages := []ChatMessage{
		{
			Role: "system",
			Content: []ContentItem{
				{
					Type: "text",
					Text: systemPrompt,
				},
			},
		},
		{
			Role: "user",
			Content: []ContentItem{
				{
					Type: "text",
					Text: userPrompt,
				},
			},
		},
	}

	req := ChatCompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   1500,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, "", TokenUsage{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, "", TokenUsage{}, fmt.Errorf("failed to create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, "", TokenUsage{}, fmt.Errorf("failed to send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(response.Body).Decode(&errorResponse); err != nil {
			return nil, "", TokenUsage{}, fmt.Errorf("OpenAI API returned non-200 status code: %d", response.StatusCode)
		}
		return nil, "", TokenUsage{}, fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, "", TokenUsage{}, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Choices) == 0 {
		return nil, "", TokenUsage{}, fmt.Errorf("no completion choices returned")
	}

	tokenUsage := TokenUsage{
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
		TotalTokens:      result.Usage.TotalTokens,
	}

	content := result.Choices[0].Message.Content
	cleanContent := cleanJSONResponse(content)

	var predictionResponse PredictionExplanationResponse
	if err := json.Unmarshal([]byte(cleanContent), &predictionResponse); err != nil {
		return nil, "", tokenUsage, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	explanations := make(map[string]FactorExplanation)
	for _, feature := range predictionResponse.Features {
		var actualFeatureName string
		var details struct {
			Value        string
			Shap         float64
			Contribution float64
			Impact       float64
		}
		var found bool

		if factorDetails, ok := factors[feature.FeatureName]; ok {
			actualFeatureName = feature.FeatureName
			details = factorDetails
			found = true
		} else {
			if mappedFeatureName, aliasExists := aliasToFeature[feature.FeatureName]; aliasExists {
				if factorDetails, ok := factors[mappedFeatureName]; ok {
					actualFeatureName = mappedFeatureName
					details = factorDetails
					found = true
				}
			}
		}

		if found {
			explanations[actualFeatureName] = FactorExplanation{
				Factor:              actualFeatureName,
				Value:               details.Value,
				Impact:              fmt.Sprintf("%.6f", details.Impact),
				Shap:                details.Shap,
				Contribution:        details.Contribution,
				ContributionPercent: fmt.Sprintf("%.2f%%", details.Contribution*100),
				Explanation:         feature.Explanation,
			}
		}
	}

	if len(explanations) == 0 {
		return nil, "", tokenUsage, fmt.Errorf("failed to parse explanations from the response. Raw content: %s", content)
	}

	return explanations, predictionResponse.Summary, tokenUsage, nil
}

func cleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	}
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}
	if strings.HasSuffix(content, "```") {
		content = strings.TrimSuffix(content, "```")
	}

	content = strings.TrimSpace(content)

	return content
}
