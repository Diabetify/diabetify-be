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
	Contribution float64
	Impact       float64
}) (string, error) {
	// Sort factors by contribution (we'll do this in the prompt)
	factorsText := ""
	for factor, details := range factors {
		impact := "Positive"
		if details.Impact < 0 {
			impact = "Negative"
		}
		factorsText += fmt.Sprintf("- %s: Impact %s (Contribution: %.3f)\n", factor, impact, details.Contribution)
	}

	prompt := fmt.Sprintf(`Analyze the following diabetes prediction results and provide a comprehensive explanation:

PREDICTION SCORE: %.1f%% risk of diabetes

CONTRIBUTING FACTORS:
%s

Please provide your analysis in the following format:

EXPLANATION_SUMMARY: [A clear, concise summary for the patient, explaining the impact of each factor]`, prediction*100, factorsText)

	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "You are a medical AI assistant specializing in diabetes risk assessment. Provide clear, actionable insights based on prediction data.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	req := ChatCompletionRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	request, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned non-200 status code: %d", response.StatusCode)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned")
	}

	return result.Choices[0].Message.Content, nil
}
