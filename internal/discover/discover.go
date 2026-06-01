package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
)

type Service struct {
	db db.DBInterface
}

type NewServiceParams struct {
	DB db.DBInterface
}

type ModelEntry struct {
	ID       string
	Provider string
}

func NewService(params NewServiceParams) *Service {
	return &Service{db: params.DB}
}

var knownProviders = []struct {
	ID        string
	BaseURL   string
	CatalogURL string
	KeyEnv    string
}{
	{"groq", "https://api.groq.com/openai/v1", "https://api.groq.com/openai/v1/models", "GROQ_API_KEY"},
	{"mistral", "https://api.mistral.ai/v1", "https://api.mistral.ai/v1/models", "MISTRAL_API_KEY"},
	{"nvidia", "https://integrate.api.nvidia.com/v1", "https://integrate.api.nvidia.com/v1/models", "NVIDIA_API_KEY"},
	{"google", "", "https://generativelanguage.googleapis.com/v1beta/models", "GOOGLE_API_KEY"},
	{"cerebras", "https://api.cerebras.ai/v1", "https://api.cerebras.ai/public/v1/models", "CEREBRAS_API_KEY"},
	{"openrouter", "https://openrouter.ai/api/v1", "https://openrouter.ai/api/v1/models", "OPENROUTER_API_KEY"},
	{"github-models", "https://models.github.ai/inference", "https://models.github.ai/catalog/models", "GITHUB_TOKEN"},
	{"opencode-zen", "https://opencode.ai/zen/v1", "https://opencode.ai/zen/v1/models", "OPENCODE_ZEN_API_KEY"},
	{"github-copilot", "https://api.githubcopilot.com", "https://api.githubcopilot.com/models", "GITHUB_TOKEN"},
}

var nonChatKeywords = []string{
	"embedding", "embed", "moderation", "ocr", "tts", "transcribe",
	"realtime", "imagen", "veo", "whisper", "speech", "dall-e",
	"stable-diffusion", "sdxl", "mistral-embed", "codestral-embed",
	"mistral-moderation", "mistral-ocr", "safety", "prompt-guard",
}

func isChatModel(id string) bool {
	lower := strings.ToLower(id)
	for _, kw := range nonChatKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}
	return true
}

func (s *Service) Discover(ctx context.Context) error {
	for _, kp := range knownProviders {
		apiKey := os.Getenv(kp.KeyEnv)
		if apiKey == "" {
			continue
		}
		entries, err := fetchCatalog(ctx, kp.ID, kp.CatalogURL, apiKey)
		if err != nil {
			fmt.Printf("  Warning [%s]: %v\n", kp.ID, err)
			continue
		}
		// Ensure provider exists in DB
		_ = s.db.UpsertProvider(&models.Provider{
			ID:         kp.ID,
			Name:       kp.ID,
			BaseURL:    kp.BaseURL,
			CatalogURL: kp.CatalogURL,
			KeyEnv:     kp.KeyEnv,
			Source:     "auto",
			Status:     "active",
			LastSynced: time.Now().Unix(),
		})
		count := 0
		for _, m := range entries {
			if !isChatModel(m.ID) {
				continue
			}
			if kp.ID == "openrouter" && !strings.HasSuffix(m.ID, ":free") {
				continue
			}
			providerID := kp.ID
			fullID := providerID + "/" + m.ID
			_ = s.db.UpsertModel(&models.Model{
				ID:          fullID,
				ProviderID:  providerID,
				DisplayName: m.ID,
				Source:      "discovered",
				Status:      "untested",
			})
			count++
		}
		fmt.Printf("  %s: %d models discovered\n", kp.ID, count)
	}
	return nil
}

func fetchCatalog(ctx context.Context, providerID, catalogURL, apiKey string) ([]ModelEntry, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", catalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	switch providerID {
	case "google":
		req.URL.RawQuery = "key=" + apiKey
	case "github-models", "github-copilot":
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	case "cerebras":
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	var entries []ModelEntry
	switch providerID {
	case "google":
		var result struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		for _, m := range result.Models {
			name := strings.TrimPrefix(m.Name, "models/")
			entries = append(entries, ModelEntry{ID: name, Provider: providerID})
		}
	default:
		var result struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		for _, m := range result.Data {
			short := m.ID
			if idx := strings.LastIndex(m.ID, "/"); idx >= 0 {
				short = m.ID[idx+1:]
			}
			entries = append(entries, ModelEntry{ID: short, Provider: providerID})
		}
	}

	return entries, nil
}

func DetectAvailableProviders() []string {
	var out []string
	for _, kp := range knownProviders {
		if os.Getenv(kp.KeyEnv) != "" {
			out = append(out, kp.ID)
		}
	}
	return out
}
