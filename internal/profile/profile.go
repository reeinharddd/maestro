package profile

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	db db.DBInterface
}

func New(database db.DBInterface) *Service {
	return &Service{db: database}
}

func (s *Service) ProfileAll(ctx context.Context, full bool) error {
	providers, err := s.db.ListProviders()
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, prov := range providers {
		if prov.Status != "active" {
			continue
		}
		prov := prov
		g.Go(func() error {
			models, err := s.db.ListModelsByProvider(prov.ID)
			if err != nil {
				return err
			}
			for _, m := range models {
				if !full && m.Status != "active" {
					continue
				}
				m := m
				prof, err := s.ProfileModel(ctx, prov, m)
				if err != nil {
					fmt.Printf("  Profile %s: %v\n", m.ID, err)
					continue
				}
				fmt.Printf("  Profile %s: stream=%v so=%v ctx=%d tps=%.1f\n",
					m.ID, prof.SupportsStream, prof.SupportsSO, prof.RealContext, prof.StreamTPS)
			}
			return nil
		})
	}
	return g.Wait()
}

func (s *Service) ProfileModel(ctx context.Context, provider models.Provider, model models.Model) (*models.ModelProfile, error) {
	apiKey := os.Getenv(provider.KeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key for %s", provider.KeyEnv)
	}

	endpoint := strings.TrimRight(provider.BaseURL, "/") + "/chat/completions"

	streamTPS, supportsStream := s.TestStreaming(endpoint, apiKey, model.DisplayName)
	supportsSO := s.TestStructuredOutput(endpoint, apiKey, model.DisplayName)
	ctxEstimate := s.EstimateContext(endpoint, apiKey, model.DisplayName)

	prof := &models.ModelProfile{
		ModelID:       model.ID,
		RealContext:   ctxEstimate,
		SupportsStream: supportsStream,
		SupportsSO:    supportsSO,
		StreamTPS:     streamTPS,
		ProfiledAt:    time.Now().Unix(),
	}

	if err := s.db.UpsertModelProfile(prof); err != nil {
		fmt.Printf("  Warning: save profile for %s: %v\n", model.ID, err)
	}

	return prof, nil
}

func (s *Service) TestStreaming(endpoint, apiKey, modelID string) (float64, bool) {
	body := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"Count from 1 to 50 slowly"}],"stream":true,"max_tokens":200}`, modelID)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader([]byte(body)))
	if err != nil {
		return 0, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, false
	}

	scanner := bufio.NewScanner(resp.Body)
	tokenCount := 0
	start := time.Now()

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") && !strings.Contains(line, "[DONE]") {
			tokenCount++
		}
		if time.Since(start) > 10*time.Second {
			break
		}
	}

	elapsed := time.Since(start).Seconds()
	if elapsed < 0.1 || tokenCount < 2 {
		return 0, false
	}

	return float64(tokenCount) / elapsed, true
}

func (s *Service) TestStructuredOutput(endpoint, apiKey, modelID string) bool {
	body := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"Say hello as JSON"}],"response_format":{"type":"json_object"},"max_tokens":100}`, modelID)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader([]byte(body)))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}
	if len(result.Choices) == 0 {
		return false
	}

	content := result.Choices[0].Message.Content
	var js json.RawMessage
	return json.Unmarshal([]byte(content), &js) == nil
}

func (s *Service) EstimateContext(endpoint, apiKey, modelID string) int {
	padding := strings.Repeat("A", 5000)
	body := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"%s"}],"max_tokens":1}`, modelID, padding)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader([]byte(body)))
	if err != nil {
		return 131072
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 131072
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return 131072
	}
	return 8192
}
