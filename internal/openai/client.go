package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	featureAliases := map[string]string{
		"age":                         "Usia",
		"bmi":                         "Indeks Massa Tubuh (BMI)",
		"brinkman_score":              "Indeks Brinkman",
		"is_hypertension":             "Hipertensi",
		"is_cholesterol":              "Kolesterol Tinggi",
		"is_bloodline":                "Riwayat Keluarga dengan Diabetes",
		"is_macrosomic_baby":          "Riwayat Melahirkan Bayi Besar",
		"smoking_status":              "Status Merokok",
		"physical_activity_frequency": "Frekuensi Aktivitas Fisik Sedang",
	}

	featureDescriptions := map[string]string{
		"age":                         "The user's age in years, represented as a whole number (e.g., 50).",
		"bmi":                         "The user's Body Mass Index (BMI), a continuous numeric value (e.g., 20.5), used to assess weight status based on Asian population classifications. BMI Classifications: < 18.5 = Underweight (Kurus), 18.5 - 22.9 = Normal, 23.0 - 24.9 = Overweight (Beresiko Obesitas), 25.0 - 29.9 = Obese I (Obesitas I), ≥ 30.0 = Obese II (Obesitas II).",
		"brinkman_score":              "Brinkman Index measures lifetime tobacco exposure: 0 = never smoked, 1 = mild smoker, 2 = moderate smoker, 3 = heavy smoker.",
		"is_hypertension":             "Indicates whether the user has been diagnosed with hypertension (high blood pressure): 0 = no, 1 = yes.",
		"is_cholesterol":              "Indicates whether the user has been diagnosed with high cholesterol: 0 = no, 1 = yes.",
		"is_bloodline":                "Indicates whether the user's parent has died due to diabetes: 0 = no, 1 = yes",
		"is_macrosomic_baby":          "Indicates whether the user has given birth to a baby weighing more than 4 kg: 0 = no, 1 = yes, 2 = not applicable (never pregnant).",
		"smoking_status":              "The user's smoking status: 0 = never smoked, 1 = former smoker, 2 = current smoker.",
		"physical_activity_frequency": "The number of days per week the user performs moderate-intensity physical activities.",
	}

	featureTable := "| Feature Name | Feature Alias | Feature Description |\n|-----|-----|-----|\n"
	for factor := range factors {
		alias := featureAliases[factor]
		description := featureDescriptions[factor]
		featureTable += fmt.Sprintf("| %s | %s | %s |\n", factor, alias, description)
	}

	shapTable := "| Feature Name | Input Value | SHAP Value |\n|-----|-----|-----|\n"
	for factor, details := range factors {
		shapTable += fmt.Sprintf("| %s | %s | %.6f |\n", factor, details.Value, details.Shap)
	}

	globalFeatureImportanceDescription := `### Global Feature Importance Analysis:
Based on the global SHAP analysis across the entire dataset, here are the key insights about feature importance for diabetes prediction:
1. **age**: This is the most significant feature. The plot clearly shows that higher age (red dots) corresponds to positive SHAP values (up to approximately +0.15), indicating a significantly higher predicted risk of diabetes. Conversely, lower age (blue dots) is associated with negative SHAP values (down to approximately -0.15), indicating a lower risk.
2. **bmi**: Body Mass Index (BMI) is the second most influential feature. High BMI values (red dots) strongly push the prediction towards a positive diabetes outcome, with SHAP values reaching as high as +0.20. Low BMI values (blue dots) have a negative impact (down to -0.10), lowering the predicted risk.
3. **is_hypertension**: Patients with hypertension (red dots) consistently have positive SHAP values (around +0.05), increasing their predicted diabetes risk. Those without hypertension (blue dots) show negative SHAP values (around -0.05), decreasing the risk.
4. **smoking_status**: Being a smoker (red dots) contributes to a higher diabetes risk with positive SHAP values (up to +0.05). Non-smoking status (blue dots) has a slightly negative impact, reducing the predicted risk.
5. **is_macrosomic_baby**: This condition (red dots) leads to positive SHAP values (up to +0.03), increasing the diabetes risk. The absence of this history (blue dots) results in a negative impact (around -0.025).
6. **brinkman_score**: This index shows that higher values (red dots) are generally associated with positive SHAP values (up to +0.05), indicating an increased risk of diabetes. Lower values (blue dots) correspond to negative SHAP values (down to -0.05), suggesting a reduced risk.
7. **is_cholesterol**: Patients with high cholesterol (red dots) have positive SHAP values (up to +0.03), increasing the predicted risk of diabetes. Normal cholesterol levels (blue dots) have a slight negative impact, lowering the risk.
8. **is_bloodline**: The presence of diabetes in the bloodline (red dots) results in positive SHAP values (up to +0.05), increasing the risk. The absence of a family history (blue dots) shows a minimal negative impact on the prediction.
7. **physical_activity_frequency**: This feature has the least impact among the top predictors. A higher frequency of physical activity (red dots) corresponds to slightly negative SHAP values (around -0.02), indicating a minor decrease in diabetes risk. Conversely, lower physical activity levels (blue dots) are associated with slightly positive SHAP values, suggesting a minor increase in risk.`

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

Provide explanations for each feature's contribution to the prediction.`, shapTable)

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
