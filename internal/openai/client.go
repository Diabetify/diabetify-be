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
	Factor       string  `json:"factor"`
	Value        string  `json:"value"`
	Impact       string  `json:"impact"`
	Shap         float64 `json:"shap"`
	Contribution float64 `json:"contribution"`
	Explanation  string  `json:"explanation"`
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
			Name:        "age",
			Alias:       "Usia",
			Description: "User's age in years (e.g., 50).",
			GlobalImportanceInsight: "Most significant feature. Higher age strongly increases diabetes risk (SHAP up to +0.15), while lower age decreases it (SHAP down to -0.15).",
			ImportanceRank: 1,
		},
		"bmi": {
			Name:        "bmi",
			Alias:       "Indeks Massa Tubuh",
			Description: "Body Mass Index (e.g., 20.5), based on Asian population classifications: <18.5=Underweight, 18.5-22.9=Normal, 23.0-24.9=Overweight, 25.0-29.9=Obese I, ≥30.0=Obese II.",
			GlobalImportanceInsight: "Second most influential feature. High BMI values strongly increase diabetes risk (SHAP up to +0.20), while low BMI values decrease it (SHAP down to -0.10).",
			ImportanceRank: 2,
		},
		"is_hypertension": {
			Name:        "is_hypertension",
			Alias:       "Hipertensi",
			Description: "Indicates if the user has hypertension (high blood pressure): 0=No, 1=Yes.",
			GlobalImportanceInsight: "A 'Yes' diagnosis increases diabetes risk (SHAP ~+0.05), while 'No' decreases it (SHAP ~-0.05).",
			ImportanceRank: 3,
		},
		"smoking_status": {
			Name:        "smoking_status",
			Alias:       "Status Merokok",
			Description: "User's smoking status: 0=Never, 1=Former, 2=Current.",
			GlobalImportanceInsight: "Being a current or former smoker increases diabetes risk (SHAP up to +0.05). Never smoking slightly decreases the risk.",
			ImportanceRank: 4,
		},
		"is_macrosomic_baby": {
			Name:        "is_macrosomic_baby",
			Alias:       "Riwayat Melahirkan Bayi Besar",
			Description: "History of giving birth to a baby over 4 kg: 0=No, 1=Yes, 2=Not applicable (never pregnant).",
			GlobalImportanceInsight: "A 'Yes' history increases diabetes risk (SHAP up to +0.03), while 'No' has a reducing effect (SHAP ~-0.025).",
			ImportanceRank: 5,
		},
		"brinkman_score": {
			Name:        "brinkman_score",
			Alias:       "Indeks Brinkman",
			Description: "Measures lifetime tobacco exposure: 0=Never, 1=Mild, 2=Moderate, 3=Heavy smoker.",
			GlobalImportanceInsight: "Higher scores increase diabetes risk (SHAP up to +0.05), while lower scores decrease it (SHAP down to -0.05).",
			ImportanceRank: 6,
		},
		"is_cholesterol": {
			Name:        "is_cholesterol",
			Alias:       "Kolesterol Tinggi",
			Description: "Indicates if the user has been diagnosed with high cholesterol: 0=No, 1=Yes.",
			GlobalImportanceInsight: "A 'Yes' diagnosis increases diabetes risk (SHAP up to +0.03). Normal cholesterol levels have a slight risk-reducing effect.",
			ImportanceRank: 7,
		},
		"is_bloodline": {
			Name:        "is_bloodline",
			Alias:       "Riwayat Keluarga dengan Diabetes",
			Description: "Indicates if a parent died from diabetes: 0=No, 1=Yes.",
			GlobalImportanceInsight: "A 'Yes' history increases diabetes risk (SHAP up to +0.05). A 'No' history has a minimal risk-reducing impact.",
			ImportanceRank: 8,
		},
		"physical_activity_frequency": {
			Name:        "physical_activity_frequency",
			Alias:       "Frekuensi Aktivitas Fisik Sedang",
			Description: "Days per week the user performs moderate-intensity physical activity.",
			GlobalImportanceInsight: "Least impactful predictor. More activity slightly decreases risk (SHAP ~-0.02), while less activity slightly increases it.",
			ImportanceRank: 9,
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
	table.WriteString("|-----|-----|-----|\n")
	
	for _, factor := range factorKeys {
		if info, exists := features[factor]; exists {
			table.WriteString(fmt.Sprintf("| %s | %s | %s |\n", 
				info.Name, info.Alias, info.Description))
		}
	}
	
	return table.String()
}

func buildGlobalImportanceDescription(features map[string]FeatureInfo) string {
	var description strings.Builder
	
	sortedFeatures := make([]FeatureInfo, 0, len(features))
	for _, info := range features {
		sortedFeatures = append(sortedFeatures, info)
	}
	
	for i := 0; i < len(sortedFeatures)-1; i++ {
		for j := 0; j < len(sortedFeatures)-i-1; j++ {
			if sortedFeatures[j].ImportanceRank > sortedFeatures[j+1].ImportanceRank {
				sortedFeatures[j], sortedFeatures[j+1] = sortedFeatures[j+1], sortedFeatures[j]
			}
		}
	}
	
	for _, info := range sortedFeatures {
		description.WriteString(fmt.Sprintf("%d. **%s**: %s\n", 
			info.ImportanceRank, info.Name, info.GlobalImportanceInsight))
	}
	
	return description.String()
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
	globalFeatureImportanceDescription := buildGlobalImportanceDescription(featureDefinitions)

	var shapTable strings.Builder
	shapTable.WriteString("| Feature Name | Input Value | SHAP Value |\n")
	shapTable.WriteString("|-----|-----|-----|\n")
	for factor, details := range factors {
		shapTable.WriteString(fmt.Sprintf("| %s | %s | %.6f |\n", 
			factor, details.Value, details.Shap))
	}

	systemPrompt := fmt.Sprintf(`### General Request:
Your job is to explain how each feature affects the user's predicted diabetes risk for the mobile app.

### How to Act:
- Act as a **medical AI explainer** for **diabetes predictions**.
- Address the user as "Anda".
- Write all explanations in **Bahasa Indonesia**.
- Use simple, everyday language that anyone can understand, especially non-experts.
- **Avoid complex terms** like SHAP, XAI, or medical jargon; if used, explain them in plain language.

### Context:
- SHAP (SHapley Additive exPlanations) is a method for explaining the output of machine learning models. SHAP shows how much each feature contributes to a specific prediction.
- Positive SHAP values (>0): "increases diabetes risk"
- Negative SHAP values (<0): "decreases diabetes risk" 

### Feature Reference:
- The table below lists each feature used in the diabetes prediction model, along with its alias and description:
%s

### Global Feature Importance:
- The list below summarizes each feature's overall importance in diabetes prediction based on global SHAP analysis (note: individual importance may vary for each user):
%s

### Output Format:
The output must be a JSON object with the following structure:
- 'summary': A 2-sentence summary that clearly explains the user's diabetes prediction based on SHAP values.
- 'features': An array with each feature's explanation. Each object must include:
    - 'feature_name': The feature's name (use the original feature name, not the alias).
    - 'explanation': The feature's role in this prediction, explained in plain, diabetes-relevant language (2 sentences).
Do not enclose the JSON in markdown code. Only return the JSON object.

### Examples:
Example 1: 
Input: BMI = 28.5, SHAP value = 0.3 
Output: "BMI Anda yang tinggi (28.5) secara signifikan meningkatkan risiko diabetes Anda. Risiko diabetes cenderung meningkat seiring dengan peningkatan indeks massa tubuh."

Example 2:
Input: Age = 45, SHAP value = 0.1 
Output: "Usia Anda yang menengah (45 tahun) secara moderat meningkatkan risiko diabetes Anda. Risiko diabetes cenderung meningkat seiring bertambahnya usia."

### Sentence Structure for Each Explanation:
- Sentence 1: State the user's specific value for this feature and whether it increases or decreases their diabetes risk based on the SHAP value direction and magnitude.
- Sentence 2: Briefly explain the general medical relationship between this feature and diabetes risk in simple terms.

### IMPORTANT: 
- Use the original feature names (like "age", "bmi", "is_hypertension") in the "feature_name" field, NOT the aliases.
- The aliases are only for display purposes in the explanation text.
`, featureTable, globalFeatureImportanceDescription)

	userPrompt := fmt.Sprintf(`Please analyze this user's diabetes prediction with the following SHAP values:

%s

Provide explanations for each feature's contribution to the prediction.`, shapTable.String())

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
		Temperature: 0.2,
		MaxTokens:   3000,
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

	content := result.Choices[0].Message.Content

	tokenUsage := TokenUsage{
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
		TotalTokens:      result.Usage.TotalTokens,
	}

	var predictionResponse PredictionExplanationResponse
	if err := json.Unmarshal([]byte(content), &predictionResponse); err != nil {
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
				Factor:       actualFeatureName,
				Value:        details.Value,
				Impact:       fmt.Sprintf("%.6f", details.Impact),
				Shap:         details.Shap,
				Contribution: details.Contribution,
				Explanation:  feature.Explanation,
			}
		}
	}

	if len(explanations) == 0 {
		return nil, "", tokenUsage, fmt.Errorf("failed to parse explanations from the response. Raw content: %s", content)
	}

	return explanations, predictionResponse.Summary, tokenUsage, nil
}
