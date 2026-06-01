package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

type providerVerify struct {
	Name      string
	KeyEnv    string
	Endpoint  string
	AuthType  string // bearer, x-api-key, query
	ModelPath string  // JSON path to model IDs in response
}

var verifyProviders = []providerVerify{
	{"OpenAI", "OPENAI_API_KEY", "https://api.openai.com/v1/models", "bearer", "data"},
	{"Anthropic", "ANTHROPIC_API_KEY", "https://api.anthropic.com/v1/models", "x-api-key", "data"},
	{"Mistral", "MISTRAL_API_KEY", "https://api.mistral.ai/v1/models", "bearer", "data"},
	{"Groq", "GROQ_API_KEY", "https://api.groq.com/openai/v1/models", "bearer", "data"},
	{"Google", "GOOGLE_API_KEY", "https://generativelanguage.googleapis.com/v1/models", "query", "models"},
	{"GitHub Models", "GITHUB_TOKEN", "https://models.inference.ai.azure.com/v1/models", "bearer", "data"},
	{"Cerebras", "CEREBRAS_API_KEY", "https://api.cerebras.ai/v1/models", "bearer", "data"},
}

func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify provider API connectivity and model list",
		Long: `Calls each provider's model list API to confirm connectivity
and compares returned model IDs against the local registry.`,
	}
	providerFlag := cmd.Flags().String("provider", "", "Only verify this provider")
	liveFlag := cmd.Flags().Bool("live", false, "Use live API calls (if false, check keys only)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
			d, err := openDB(nil)
		if err != nil {
			return err
		}
		defer d.Close()

		providers := verifyProviders
		if *providerFlag != "" {
			filtered := []providerVerify{}
			for _, p := range providers {
				if strings.EqualFold(p.Name, *providerFlag) {
					filtered = append(filtered, p)
					break
				}
			}
			if len(filtered) == 0 {
				return fmt.Errorf("unknown provider: %s", *providerFlag)
			}
			providers = filtered
		}

		type result struct {
			Provider   string
			KeyStatus  string
			APIModels  int
			APIFailed  bool
			APIError   string
		}

		results := make([]result, len(providers))
		var wg sync.WaitGroup

		for i, p := range providers {
			wg.Add(1)
			go func(i int, p providerVerify) {
				defer wg.Done()
				r := result{Provider: p.Name}

				key := os.Getenv(p.KeyEnv)
				if key == "" {
					envFile, _ := parseEnvFile(defaultEnvPath())
					if v, ok := envFile[p.KeyEnv]; ok {
						key = v
					}
				}

				if key == "" {
					r.KeyStatus = "MISSING"
				} else {
					r.KeyStatus = "present"
				}

				if *liveFlag && r.KeyStatus == "present" {
					models, err := callProviderAPI(p, key)
					if err != nil {
						r.APIFailed = true
						r.APIError = err.Error()
					} else {
						r.APIModels = len(models)
					}
				}

				results[i] = r
			}(i, p)
		}
		wg.Wait()

		fmt.Println("=== Provider Verification ===")
		fmt.Println()
		for _, r := range results {
			status := r.KeyStatus
			if r.APIFailed {
				status += fmt.Sprintf(", API: FAIL (%s)", r.APIError)
			} else if *liveFlag && r.KeyStatus == "present" {
				status += fmt.Sprintf(", API: OK (%d models)", r.APIModels)
			}
			fmt.Printf("  %-15s  %s\n", r.Provider+":", status)
		}
		fmt.Println()
		fmt.Println("Tip: use --live to actually call each provider's API")

		return nil
	}

	return cmd
}

func defaultEnvPath() string {
	return OpenCodeEnvPath()
}

func callProviderAPI(p providerVerify, key string) ([]string, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", p.Endpoint, nil)
	if err != nil {
		return nil, err
	}

	switch p.AuthType {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+key)
	case "x-api-key":
		req.Header.Set("x-api-key", key)
	case "query":
		q := req.URL.Query()
		q.Set("key", key)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	var modelIDs []string
	switch p.ModelPath {
	case "data":
		if data, ok := raw["data"].([]interface{}); ok {
			for _, item := range data {
				if m, ok := item.(map[string]interface{}); ok {
					if id, ok := m["id"].(string); ok {
						modelIDs = append(modelIDs, id)
					}
				}
			}
		}
	case "models":
		if models, ok := raw["models"].([]interface{}); ok {
			for _, item := range models {
				if m, ok := item.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok {
						modelIDs = append(modelIDs, name)
					}
				}
			}
		}
	}

	return modelIDs, nil
}
