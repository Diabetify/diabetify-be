package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type ContentItem struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
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
	Name        string
	Alias       string
	Description string
}

func getFeatureDefinitions() map[string]FeatureInfo {
	return map[string]FeatureInfo{
		"age": {
			Name:        "age",
			Alias:       "Usia",
			Description: "User's age in years (e.g., 50).",
		},
		"bmi": {
			Name:        "bmi",
			Alias:       "Indeks Massa Tubuh",
			Description: "Body Mass Index (e.g., 20.5), based on Asian population classifications: <18.5=Underweight, 18.5-22.9=Normal, 23.0-24.9=Overweight, 25.0-29.9=Obese I, ≥30.0=Obese II.",
		},
		"is_hypertension": {
			Name:        "is_hypertension",
			Alias:       "Hipertensi",
			Description: "Indicates if the user has hypertension (high blood pressure): 0=No, 1=Yes.",
		},
		"smoking_status": {
			Name:        "smoking_status",
			Alias:       "Status Merokok",
			Description: "User's smoking status: 0=Never smoked, 1=Former smoker, 2=Current smoker.",
		},
		"is_macrosomic_baby": {
			Name:        "is_macrosomic_baby",
			Alias:       "Riwayat Melahirkan Bayi Besar",
			Description: "History of giving birth to a baby over 4 kg: 0=No, 1=Yes, 2=Not applicable (never pregnant).",
		},
		"brinkman_score": {
			Name:        "brinkman_score",
			Alias:       "Indeks Brinkman",
			Description: "Measures lifetime tobacco exposure: 0=Never smoked, 1=Mild smoker, 2=Moderate smoker, 3=Heavy smoker.",
		},
		"is_cholesterol": {
			Name:        "is_cholesterol",
			Alias:       "Kolesterol Tinggi",
			Description: "Indicates if the user has been diagnosed with high cholesterol: 0=No, 1=Yes.",
		},
		"is_bloodline": {
			Name:        "is_bloodline",
			Alias:       "Riwayat Keluarga dengan Diabetes",
			Description: "Indicates if a parent died from diabetes: 0=No, 1=Yes.",
		},
		"physical_activity_frequency": {
			Name:        "physical_activity_frequency",
			Alias:       "Frekuensi Aktivitas Fisik Sedang",
			Description: "Days per week the user performs moderate-intensity physical activity.",
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
	table.WriteString("|--------------|---------------|--------------------|\n")

	for _, factor := range factorKeys {
		if info, exists := features[factor]; exists {
			table.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				info.Name, info.Alias, info.Description))
		}
	}

	return table.String()
}

func imageToBase64(imagePath string) (string, error) {
	imageData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %v", err)
	}

	base64String := base64.StdEncoding.EncodeToString(imageData)
	return base64String, nil
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

	base64Image, err := imageToBase64("../internal/openai/source/global.png")
	if err != nil {
		return nil, "", TokenUsage{}, fmt.Errorf("failed to load global importance image: %v", err)
	}

	var shapTable strings.Builder
	shapTable.WriteString("| Feature Name | Input Value | SHAP Value |\n")
	shapTable.WriteString("|--------------|-------------|------------|\n")
	for factor, details := range factors {
		shapTable.WriteString(fmt.Sprintf("| %s | %s | %.6f |\n",
			factor, details.Value, details.Shap))
	}

	systemPrompt := fmt.Sprintf(`# Diabetes Prediction Explanation System
	
## 1. KNOWLEDGE SOURCE CONTEXT
### SHAP Value Interpretation
- **SHAP (SHapley Additive exPlanations)**: A method for explaining machine learning model outputs
- **Positive SHAP values (>0)**: Feature increases diabetes risk for this individual
- **Negative SHAP values (<0)**: Feature decreases diabetes risk for this individual
- **Magnitude**: Larger absolute values indicate stronger influence on the prediction

### Feature Knowledge Base
The diabetes prediction model uses the following features with their clinical significance:

%s

### Global Feature Impact Analysis
The image provided shows the global SHAP value distribution for each feature across the entire dataset. This chart demonstrates:
- **Feature value ranges**: Color gradient (blue to red) indicates low to high values
- **SHAP impact patterns**: How feature values affect diabetes risk
- **Risk contribution trends**: Positive SHAP = increased risk; negative SHAP = decreased risk
- **Feature importance**: Wider SHAP spread = greater impact on prediction
- **Value-specific effects**: Colors reveal how specific values relate to risk changes

Use this chart to understand how the user's specific feature values compare to the global patterns and explain why their values contribute to risk increase or decrease.

## 2. HOW TO ACT:
### Role and Persona
- Act as a **medical AI explainer** specialist focusing on **diabetes risk prediction models**

### Communication Style
- Address the user as "Anda" (formal Indonesian)
- Write all explanations in **Bahasa Indonesia**
- Use **simple, everyday language** suitable for non-expert audiences
- Avoid technical jargon (SHAP, XAI, machine learning terms) without explanation
- When technical terms are necessary, provide clear, accessible definitions

### Explanation Approach
- Focus on **what the data shows** rather than medical interpretations
- Describe **data patterns and correlations** objectively
- Reference the global SHAP patterns shown in the chart to explain individual predictions

## 3. GENERAL REQUEST:
Your primary task is to generate personalized explanations for diabetes risk predictions based on SHAP values.

## 4. OUTPUT FORMAT:
### JSON Structure Requirements
The output must be a **valid JSON object** with the following structure:

**JSON Format:**
{
  "summary": "string",
  "features": [
    {
      "feature_name": "string", 
      "explanation": "string"
    }
  ]
}

### Field Specifications
- **'summary'**: A concise 2-sentence summary explaining the overall diabetes risk based on SHAP analysis
- **'features'**: Array containing explanations for each feature
  - **'feature_name'**: Use the original feature name (e.g., "age", "bmi", "is_hypertension"), NOT the Indonesian alias
  - **'explanation'**: Two-sentence explanation following the specified structure

### Explanation Structure Rules
- **Sentence 1**: State the user's specific value and its directional impact (increase/decrease) on diabetes risk prediction
- **Sentence 2**: Explain why this occurs by describing the general statistical relationship between the feature and diabetes risk using natural, contrastive language

### Formatting Rules
- **No markdown code blocks** around the JSON output
- Return **only the JSON object**
- Ensure valid JSON syntax

## 5. FEW-SHOT EXAMPLES
### Example 1: High BMI Impact
**Input Data**: BMI = 28.5, SHAP value = +0.30
**Expected Output**: 
"explanation": "BMI Anda yang tinggi (28.5) secara signifikan meningkatkan risiko diabetes Anda. BMI yang tinggi cenderung meningkatkan risiko diabetes, sedangkan BMI yang normal cenderung menurunkan risiko diabetes.

### Example 2: Young Age Factor
**Input Data**: Age = 25, SHAP value = -0.10
**Expected Output**:
"explanation": "Usia muda Anda (25 tahun) menurunkan risiko diabetes Anda. Usia muda cenderung menurunkan risiko diabetes, sedangkan usia tua cenderung meningkatkan risiko diabetes."

## 6. IMPORTANT GUIDELINES
### Technical Requirements
- **Feature Naming**: Always use original English feature names in the "feature_name" field
- **Aliases**: Indonesian aliases are for explanation text only, not for field names
- **Chart Understanding**: While you should not mention the chart explicitly, use your understanding of the global SHAP patterns to inform your explanations
`, featureTable)

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
				{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
					},
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
