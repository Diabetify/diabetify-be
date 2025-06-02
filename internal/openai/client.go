package openai

import (
	"bytes"
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

type FactorExplanation struct {
	Factor       string  `json:"factor"`
	Value        string  `json:"value"`
	Impact       string  `json:"impact"`
	Contribution float64 `json:"contribution"`
	Explanation  string  `json:"explanation"`
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
	Contribution float64
	Impact       float64
}) (map[string]FactorExplanation, error) {
	factorsText := ""
	for factor, details := range factors {
		impact := "Positive"
		if details.Impact < 0 {
			impact = "Negative"
		}
		factorsText += fmt.Sprintf("- [%s]: Value: %s, Impact %s (Contribution: %.3f)\n", factor, details.Value, impact, details.Contribution)
	}

	prompt := fmt.Sprintf(`Anda adalah asisten AI medis. Analisis faktor-faktor prediksi diabetes berikut dan berikan penjelasan rinci untuk setiap faktor dalam Bahasa Indonesia.

PREDICTION SCORE: %.1f%% risiko diabetes

FAKTOR-FAKTOR YANG BERKONTRIBUSI:
%s

Untuk setiap faktor, berikan respons dalam format berikut:

[NAMA_FAKTOR]:
Penjelasan: [2-3 kalimat dalam Bahasa Indonesia yang menjelaskan dampak faktor ini terhadap risiko diabetes]

Mulai dengan faktor kontribusi yang paling besar.`, prediction*100, factorsText)

	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "Anda adalah asisten AI medis yang mengkhususkan diri dalam penilaian risiko diabetes. Berikan penjelasan yang jelas untuk setiap faktor dalam Bahasa Indonesia sesuai format yang ditentukan.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	req := ChatCompletionRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   2000,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	request, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(response.Body).Decode(&errorResponse); err != nil {
			return nil, fmt.Errorf("OpenAI API returned non-200 status code: %d", response.StatusCode)
		}
		return nil, fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	content := result.Choices[0].Message.Content

	explanations := make(map[string]FactorExplanation)

	sections := strings.Split(content, "\n\n")

	for _, section := range sections {
		if section == "" {
			continue
		}
		lines := strings.Split(strings.TrimSpace(section), "\n")
		if len(lines) < 2 {
			continue
		}

		factorLine := strings.TrimSpace(lines[0])
		if !strings.HasPrefix(factorLine, "[") || !strings.Contains(factorLine, "]:") {
			continue
		}

		factor := strings.TrimSuffix(strings.TrimPrefix(factorLine, "["), "]:")
		if factor == "" {
			continue
		}

		explanationLine := strings.TrimSpace(lines[1])
		if !strings.HasPrefix(strings.ToLower(explanationLine), "penjelasan:") {
			continue
		}

		explanation := strings.TrimSpace(strings.TrimPrefix(explanationLine, "Penjelasan:"))

		currentFactor := FactorExplanation{
			Factor:      factor,
			Explanation: explanation,
		}

		if details, ok := factors[factor]; ok {
			currentFactor.Contribution = details.Contribution
			explanations[factor] = currentFactor
		}
	}

	if len(explanations) == 0 {
		return nil, fmt.Errorf("failed to parse any explanations from the response. Raw content: %s", content)
	}

	return explanations, nil
}
