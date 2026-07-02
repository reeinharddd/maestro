package credentials

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type BitwardenStore struct{}

func NewBitwardenStore(opts map[string]string) (*BitwardenStore, error) {
	if _, err := exec.LookPath("bw"); err != nil {
		return nil, fmt.Errorf("bw CLI not found in PATH: %w", err)
	}
	return &BitwardenStore{}, nil
}

func (bs *BitwardenStore) getSession(ctx context.Context) (string, error) {
	if s := os.Getenv("BW_SESSION"); s != "" {
		return s, nil
	}
	cmd := exec.CommandContext(ctx, "bw", "unlock", "--raw")
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bw unlock failed (set BW_SESSION env var): %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (bs *BitwardenStore) bwCmd(ctx context.Context, session string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "bw", args...)
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)
	return cmd
}

func (bs *BitwardenStore) Get(ctx context.Context, service, key string) (string, error) {
	session, err := bs.getSession(ctx)
	if err != nil {
		return "", err
	}
	cmd := bs.bwCmd(ctx, session, "get", "item", service)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bw get item %s: %w", service, err)
	}
	var item struct {
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(output, &item); err != nil {
		return "", fmt.Errorf("parse bw item: %w", err)
	}
	for _, f := range item.Fields {
		if f.Name == key {
			return f.Value, nil
		}
	}
	return "", nil
}

func (bs *BitwardenStore) Set(ctx context.Context, service, key, value string) error {
	session, err := bs.getSession(ctx)
	if err != nil {
		return err
	}
	existing, _ := bs.List(ctx, service)
	if len(existing) > 0 {
		return bs.updateField(ctx, session, service, key, value)
	}
	return bs.createItem(ctx, session, service, key, value)
}

func (bs *BitwardenStore) updateField(ctx context.Context, session, service, key, value string) error {
	// Get the full item from Bitwarden to get its ID and structure
	getCmd := bs.bwCmd(ctx, session, "get", "item", service)
	raw, err := getCmd.Output()
	if err != nil {
		return fmt.Errorf("bw get item %s: %w", service, err)
	}

	// Parse item to get fields and id
	var item struct {
		ID     string `json:"id"`
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  int    `json:"type"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return fmt.Errorf("parse bw item: %w", err)
	}

	// Update existing field or append new one
	found := false
	for i := range item.Fields {
		if item.Fields[i].Name == key {
			item.Fields[i].Value = value
			found = true
			break
		}
	}
	if !found {
		item.Fields = append(item.Fields, struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  int    `json:"type"`
		}{Name: key, Value: value, Type: 0})
	}

	// Re-serialize only the fields portion we need to update
	patch := struct {
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  int    `json:"type"`
		} `json:"fields"`
	}{Fields: item.Fields}
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal patch: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(patchJSON)
	editCmd := bs.bwCmd(ctx, session, "edit", "item", item.ID, encoded)
	out, err := editCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bw edit item %s: %s", item.ID, string(out))
	}
	return nil
}

func (bs *BitwardenStore) createItem(ctx context.Context, session, service, key, value string) error {
	data := fmt.Sprintf(
		`{"type":1,"name":"%s","login":{"username":"%s"},"fields":[{"name":"%s","value":"%s","type":0}]}`,
		service, service, key, value,
	)
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	cmd := bs.bwCmd(ctx, session, "create", "item", encoded)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bw create: %s", string(out))
	}
	return nil
}

func (bs *BitwardenStore) Delete(ctx context.Context, service, key string) error {
	session, err := bs.getSession(ctx)
	if err != nil {
		return err
	}

	// Get full item to get its ID
	getCmd := bs.bwCmd(ctx, session, "get", "item", service)
	raw, err := getCmd.Output()
	if err != nil {
		return fmt.Errorf("bw get item %s: %w", service, err)
	}

	// Parse and remove the field
	var item struct {
		ID     string `json:"id"`
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  int    `json:"type"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return fmt.Errorf("parse bw item: %w", err)
	}

	// Filter out the key to remove
	var updatedFields []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
		Type  int    `json:"type"`
	}
	for _, f := range item.Fields {
		if f.Name != key {
			updatedFields = append(updatedFields, f)
		}
	}

	patch := struct {
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  int    `json:"type"`
		} `json:"fields"`
	}{Fields: updatedFields}
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal patch: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(patchJSON)
	editCmd := bs.bwCmd(ctx, session, "edit", "item", item.ID, encoded)
	return editCmd.Run()
}

func (bs *BitwardenStore) List(ctx context.Context, service string) (map[string]string, error) {
	session, err := bs.getSession(ctx)
	if err != nil {
		return nil, err
	}
	cmd := bs.bwCmd(ctx, session, "get", "item", service)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bw get item %s: %w", service, err)
	}
	var item struct {
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(output, &item); err != nil {
		return nil, fmt.Errorf("parse bw item: %w", err)
	}
	result := make(map[string]string)
	for _, f := range item.Fields {
		result[f.Name] = f.Value
	}
	return result, nil
}

func (bs *BitwardenStore) Test(ctx context.Context, service string) error {
	cmd := exec.CommandContext(ctx, "bw", "status")
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("bw not accessible: %w", err)
	}
	var status struct {
		Status string `json:"status"`
		Email  string `json:"userEmail"`
	}
	if err := json.Unmarshal(output, &status); err != nil {
		return fmt.Errorf("parse bw status: %w", err)
	}
	if status.Status == "locked" {
		return fmt.Errorf("vault is locked - run 'bw unlock --raw' and set BW_SESSION")
	}
	fmt.Printf("  bw status: %s (user: %s)\n", status.Status, status.Email)
	return nil
}

func (bs *BitwardenStore) Name() string {
	return "bitwarden"
}
