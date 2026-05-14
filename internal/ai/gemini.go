package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/execbrief/backend/internal/models"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// analysisPrompt instructs Gemini to return strict JSON without any markdown wrapper.
const analysisPrompt = `You are an expert meeting analyst and executive assistant. Analyze the meeting transcript below and extract structured intelligence.

STRICT OUTPUT RULES (read carefully):
- Output ONLY a single raw JSON object. No markdown. No code fences. No backticks. No explanation before or after.
- Your entire response must start with { and end with }
- Do NOT wrap the JSON in ` + "`" + `json` + "`" + ` or any other block.
- Do not invent owners — use null if a person is not mentioned by name.
- Do not invent due dates — use null if not explicitly stated.
- Use null for any unknown field value.
- Extract ONLY information grounded in the transcript text. No assumptions.
- Executive summary must be strategic and CEO-focused.
- Recommended CEO actions must be practical and directly supported by transcript evidence.
- Every array must be present (use [] if there are no items).

REQUIRED JSON STRUCTURE (return exactly this shape):

{
  "normal_summary": "string — concise paragraph summarizing the meeting",
  "executive_summary": {
    "accomplishments": ["string"],
    "blockers": ["string"],
    "pending_tasks": ["string"],
    "risks": ["string"],
    "recommended_ceo_actions": ["string"]
  },
  "action_items": [
    {
      "task": "string",
      "owner": "string or null",
      "due_date": "string or null",
      "priority": "low|medium|high or null",
      "status": "pending",
      "source_quote": "string or null"
    }
  ],
  "decisions": [
    {
      "decision": "string",
      "owner": "string or null",
      "source_quote": "string or null"
    }
  ]
}

MEETING TRANSCRIPT:
%s`

// jsonExtract finds the outermost JSON object in a string,
// handling cases where the model wraps it in markdown despite instructions.
var jsonObjectRE = regexp.MustCompile(`(?s)\{.*\}`)

type GeminiClient struct {
	client    *genai.Client
	modelName string
}

func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	return &GeminiClient{
		client: client,
		// gemini-2.5-flash: stable, widely available, supports JSON-mode reliably.
		// gemini-2.0-flash requires ResponseSchema alongside ResponseMIMEType which
		// would add significant boilerplate; stick with 1.5-flash for the MVP.
		modelName: "gemini-2.5-flash",
	}, nil
}

func (g *GeminiClient) Close() error {
	return g.client.Close()
}

func (g *GeminiClient) AnalyzeTranscript(ctx context.Context, transcript string) (*models.AIResponse, map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	if !isLikelyText(transcript) {
		return nil, nil, fmt.Errorf("transcript appears to contain binary data. Please upload a plain text file (.txt, .vtt, or .srt)")
	}

	model := g.client.GenerativeModel(g.modelName)
	model.SetTemperature(0.1)
	// Do NOT set ResponseMIMEType here — gemini-2.0-flash rejects JSON mode
	// without a ResponseSchema, and gemini-2.5-flash produces cleaner results
	// when guided purely through the prompt.

	prompt := fmt.Sprintf(analysisPrompt, transcript)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("[Gemini] GenerateContent error for model %s: %v", g.modelName, err)
		return nil, nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, nil, fmt.Errorf("Gemini returned no candidates — the transcript may have been filtered")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		reason := ""
		if candidate.FinishReason != 0 {
			reason = fmt.Sprintf(" (finish reason: %v)", candidate.FinishReason)
		}
		return nil, nil, fmt.Errorf("Gemini returned an empty response%s", reason)
	}

	rawText := fmt.Sprintf("%v", candidate.Content.Parts[0])
	log.Printf("[Gemini] raw response length: %d chars", len(rawText))

	jsonText := extractJSON(rawText)
	if jsonText == "" {
		return nil, nil, fmt.Errorf("could not find JSON object in Gemini response (raw preview: %s)", truncate(rawText, 300))
	}

	var aiResponse models.AIResponse
	if err := json.Unmarshal([]byte(jsonText), &aiResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to parse AI JSON: %w (raw: %s)", err, truncate(jsonText, 300))
	}

	var rawMap map[string]interface{}
	_ = json.Unmarshal([]byte(jsonText), &rawMap)

	return &aiResponse, rawMap, nil
}

// extractJSON strips markdown fences and returns the outermost JSON object.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Strip ```json ... ``` or ``` ... ```
	for _, fence := range []string{"```json", "```"} {
		if strings.HasPrefix(s, fence) {
			s = strings.TrimPrefix(s, fence)
			s = strings.TrimSuffix(s, "```")
			s = strings.TrimSpace(s)
			break
		}
	}

	// If the string already starts with {, use it directly
	if strings.HasPrefix(s, "{") {
		return s
	}

	// Otherwise try to extract the first JSON object via regex
	if m := jsonObjectRE.FindString(s); m != "" {
		return m
	}
	return ""
}

// isLikelyText returns false if the content looks like binary data.
func isLikelyText(s string) bool {
	if len(s) == 0 {
		return false
	}
	sample := s
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	nullCount, controlCount := 0, 0
	for _, r := range sample {
		if r == 0 {
			nullCount++
		} else if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}
	}
	sampleLen := len(sample)
	return nullCount <= sampleLen/100 && controlCount <= sampleLen/20
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
