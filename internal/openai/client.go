package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	Description string `json:"description"`
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

func (c *Client) GeneratePredictionExplanation(prediction float64, factors map[string]struct {
	Value        string
	Shap         float64
	Contribution float64
	Impact       float64
}) (map[string]FactorExplanation, string, TokenUsage, error) {
	featureAliases := map[string]string{
		"age":                         "Usia",
		"bmi":                         "BMI (Body Mass Index)",
		"brinkman_score":              "Indeks Brinkman",
		"is_hypertension":             "Hipertensi",
		"is_cholesterol":              "Kolesterol",
		"is_bloodline":                "Riwayat Keluarga Diabetes",
		"is_macrosomic_baby":          "Riwayat Bayi Makrosomik",
		"smoking_status":              "Status Merokok",
		"physical_activity_frequency": "Frekuensi Aktivitas Fisik",
	}

	featureDescriptions := map[string]string{
		"age":                         "Usia pengguna dalam tahun",
		"bmi":                         "Indeks Massa Tubuh yang dihitung dari berat dan tinggi badan",
		"brinkman_score":              "Indeks yang menggambarkan paparan rokok sepanjang hidup",
		"is_hypertension":             "Apakah pengguna memiliki tekanan darah tinggi",
		"is_cholesterol":              "Apakah pengguna memiliki masalah kolesterol",
		"is_bloodline":                "Apakah ada riwayat diabetes dalam keluarga",
		"is_macrosomic_baby":          "Apakah pernah melahirkan bayi dengan berat > 4kg",
		"smoking_status":              "Status merokok pengguna saat ini",
		"physical_activity_frequency": "Seberapa sering pengguna melakukan aktivitas fisik per minggu",
	}

	featureTable := "| Feature Name | Feature Alias | Feature Description |\n|-----|-----|-----|\n"
	for factor := range factors {
		alias := featureAliases[factor]
		description := featureDescriptions[factor]
		featureTable += fmt.Sprintf("| %s | %s | %s |\n", factor, alias, description)
	}

	bmiTable := "| BMI Range (kg/m²) | Classification |\n|-----|-----|\n"
	bmiTable += "| < 18.5 | Underweight (Kurus) |\n"
	bmiTable += "| 18.5 - 24.9 | Normal |\n"
	bmiTable += "| 25.0 - 29.9 | Overweight (Gemuk) |\n"
	bmiTable += "| ≥ 30.0 | Obese (Obesitas) |\n"

	shapTable := "| Feature Name | Input Value | SHAP Value |\n|-----|-----|-----|\n"
	for factor, details := range factors {
		shapTable += fmt.Sprintf("| %s | %s | %.6f |\n", factor, details.Value, details.Shap)
	}

	prompt := fmt.Sprintf(`### 1. Context:
- SHAP (SHapley Additive exPlanations) is a method for explaining the output of machine learning models. SHAP shows how much each feature contributes to a specific prediction.
- The following table lists the dataset's feature names, their aliases, and descriptions:
%s

- The following table is a BMI classification reference:
%s

- The following table contains the input values and SHAP values for this specific user:
%s

### 2. General Request:
Your job is to explain the contribution of each feature to this user's predicted diabetes risk.

### 3. How to Act:
- You are acting as a **medical AI explainer** for **diabetes predictions.**
- Address the user as "Anda".
- All explanations **must be written in Bahasa Indonesia.**
- Use simple, everyday language that can be easily understood by non-experts.
- Each feature explanation **must be 2-3 sentences maximum.**

### 4. Output Format:
The output must be a JSON object with the following structure:
- 'summary': A summary that gives an easy-to-understand explanation of the user's diabetes prediction result based on the SHAP values.
- An explanation for each feature's contribution in a JSON array called 'features'. Each object must have:
    - 'feature_name': the feature name
    - 'description': your interpreted description of this feature
    - 'explanation': the feature's role in this prediction, explained in plain language with any relevant diabetes-specific context.
Do not enclose the JSON in markdown code. Only return the JSON object.`, featureTable, bmiTable, shapTable)

	messages := []ChatMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	req := ChatCompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   5000,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, "", TokenUsage{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	request, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
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
		return nil, "", tokenUsage, fmt.Errorf("failed to parse any explanations from the response. Raw content: %s", content)
	}

	return explanations, predictionResponse.Summary, tokenUsage, nil
}
