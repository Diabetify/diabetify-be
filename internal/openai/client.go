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
			Description: "The user's age in years, represented as a whole number (e.g., 50).",
			GlobalImportanceInsight: "This is the most significant feature. Higher age corresponds to positive SHAP values (up to approximately +0.15), indicating significantly higher predicted risk of diabetes. Lower age is associated with negative SHAP values (down to approximately -0.15), indicating lower risk.",
			ImportanceRank: 1,
		},
		"bmi": {
			Name:        "bmi",
			Alias:       "Indeks Massa Tubuh (BMI)",
			Description: "The user's Body Mass Index (BMI), a continuous numeric value (e.g., 20.5), used to assess weight status based on Asian population classifications. BMI Classifications: < 18.5 = Underweight (Kurus), 18.5 - 22.9 = Normal, 23.0 - 24.9 = Overweight (Beresiko Obesitas), 25.0 - 29.9 = Obese I (Obesitas I), ≥ 30.0 = Obese II (Obesitas II).",
			GlobalImportanceInsight: "Body Mass Index is the second most influential feature. High BMI values strongly push the prediction towards positive diabetes outcome, with SHAP values reaching as high as +0.20. Low BMI values have negative impact (down to -0.10), lowering the predicted risk.",
			ImportanceRank: 2,
		},
		"is_hypertension": {
			Name:        "is_hypertension",
			Alias:       "Hipertensi",
			Description: "Indicates whether the user has been diagnosed with hypertension (high blood pressure): 0 = no, 1 = yes.",
			GlobalImportanceInsight: "Patients with hypertension consistently have positive SHAP values (around +0.05), increasing their predicted diabetes risk. Those without hypertension show negative SHAP values (around -0.05), decreasing the risk.",
			ImportanceRank: 3,
		},
		"smoking_status": {
			Name:        "smoking_status",
			Alias:       "Status Merokok",
			Description: "The user's smoking status: 0 = never smoked, 1 = former smoker, 2 = current smoker.",
			GlobalImportanceInsight: "Being a smoker contributes to higher diabetes risk with positive SHAP values (up to +0.05). Non-smoking status has slightly negative impact, reducing the predicted risk.",
			ImportanceRank: 4,
		},
		"is_macrosomic_baby": {
			Name:        "is_macrosomic_baby",
			Alias:       "Riwayat Melahirkan Bayi Besar",
			Description: "Indicates whether the user has given birth to a baby weighing more than 4 kg: 0 = no, 1 = yes, 2 = not applicable (never pregnant).",
			GlobalImportanceInsight: "This condition leads to positive SHAP values (up to +0.03), increasing diabetes risk. The absence of this history results in negative impact (around -0.025).",
			ImportanceRank: 5,
		},
		"brinkman_score": {
			Name:        "brinkman_score",
			Alias:       "Indeks Brinkman",
			Description: "Brinkman Index measures lifetime tobacco exposure: 0 = never smoked, 1 = mild smoker, 2 = moderate smoker, 3 = heavy smoker.",
			GlobalImportanceInsight: "Higher values are generally associated with positive SHAP values (up to +0.05), indicating increased risk of diabetes. Lower values correspond to negative SHAP values (down to -0.05), suggesting reduced risk.",
			ImportanceRank: 6,
		},
		"is_cholesterol": {
			Name:        "is_cholesterol",
			Alias:       "Kolesterol Tinggi",
			Description: "Indicates whether the user has been diagnosed with high cholesterol: 0 = no, 1 = yes.",
			GlobalImportanceInsight: "Patients with high cholesterol have positive SHAP values (up to +0.03), increasing predicted risk of diabetes. Normal cholesterol levels have slight negative impact, lowering the risk.",
			ImportanceRank: 7,
		},
		"is_bloodline": {
			Name:        "is_bloodline",
			Alias:       "Riwayat Keluarga dengan Diabetes",
			Description: "Indicates whether the user's parent has died due to diabetes: 0 = no, 1 = yes",
			GlobalImportanceInsight: "The presence of diabetes in the bloodline results in positive SHAP values (up to +0.05), increasing the risk. The absence of family history shows minimal negative impact on the prediction.",
			ImportanceRank: 8,
		},
		"physical_activity_frequency": {
			Name:        "physical_activity_frequency",
			Alias:       "Frekuensi Aktivitas Fisik Sedang",
			Description: "The number of days per week the user performs moderate-intensity physical activities.",
			GlobalImportanceInsight: "This feature has the least impact among the top predictors. Higher frequency of physical activity corresponds to slightly negative SHAP values (around -0.02), indicating minor decrease in diabetes risk. Lower physical activity levels are associated with slightly positive SHAP values, suggesting minor increase in risk.",
			ImportanceRank: 9,
		},
	}
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
	description.WriteString("### Global Feature Importance Analysis:\n")
	description.WriteString("Based on the global SHAP analysis across the entire dataset, here are the key insights about feature importance for diabetes prediction:\n")
	
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
Your job is to explain the contribution of each feature to this user's predicted diabetes risk.

### How to Act:
- You are acting as a **medical AI explainer** for **diabetes predictions.**
- Address the user as "Anda".
- All explanations **must be written in Bahasa Indonesia.**
- Use simple, everyday language that's easy for everyone to understand, especially people who **don't have a background in medicine or technology**.
- **Avoid using complex terms** like SHAP, XAI, or medical jargon. If such terms must be used, explain them in a way that a **regular person can easily grasp**.

### Context:
- SHAP (SHapley Additive exPlanations) is a method for explaining the output of machine learning models. SHAP shows how much each feature contributes to a specific prediction.

### Feature Reference:
- The following table lists the features used in the diabetes prediction model, along with their aliases and descriptions:
%s

%s

### Output Format:
The output must be a JSON object with the following structure:
- 'summary': A summary that gives an easy-to-understand explanation of the user's diabetes prediction result based on the SHAP values.
- An explanation for each feature's contribution in a JSON array called 'features'. Each object must have:
    - 'feature_name': the feature name
    - 'explanation': the feature's role in this prediction, explained in plain language with any relevant diabetes-specific context. Each explanation must be 2 sentences long.
- Sentence Structure Breakdown (Per Explanation):
		- Sentence 1: States the condition and its influence on diabetes risk (positive or negative). It explains whether a certain factor increases or decreases the person's diabetes risk based on their data.
		- Sentence 2: Provides general background or reasoning about how or why that factor affects diabetes risk. This sentence gives a simple explanation or context that connects the factor to diabetes risk based on Global Feature Importance Analysis.
- This is an example of a good feature explanation: '(Value) yang (adjective) secara (adverb) (verb) risiko diabetes Anda. Risiko diabetes cenderung (verb) seiring (noun phrase) (this factor).'

Do not enclose the JSON in markdown code. Only return the JSON object.`, featureTable, globalFeatureImportanceDescription)

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
		Temperature: 0.3,
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
		if details, ok := factors[feature.FeatureName]; ok {
			explanations[feature.FeatureName] = FactorExplanation{
				Factor:       feature.FeatureName,
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
