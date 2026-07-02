package credentials

import (
	"context"
	"encoding/json"
	"fmt"
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
	cmd := exec.CommandContext(ctx, "bw", "unlock", "--raw")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bw unlock failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (bs *BitwardenStore) runBw(ctx context.Context, session string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "bw", args...)
	if session != "" {
		cmd.Env = append(cmd.Env, "BW_SESSION="+session)
	}
	return cmd.Output()
}

func (bs *BitwardenStore) Get(ctx context.Context, service, key string) (string, error) {
	session, err := bs.getSession(ctx)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "bw", "get", "item", service)
	cmd.Env = append(cmd.Env, "BW_SESSION="+session)
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
	existing, _ := bs.Get(ctx, service, key)
	if existing != "" {
		cmd := exec.CommandContext(ctx, "bw", "edit", "item", service, "--field", key, "--value", value)
		cmd.Env = append(cmd.Env, "BW_SESSION="+session)
		return cmd.Run()
	}
	cmd := exec.CommandContext(ctx, "bw", "create", "item", "login", "--name", service, "--field", key+"="+value)
	cmd.Env = append(cmd.Env, "BW_SESSION="+session)
	return cmd.Run()
}

func (bs *BitwardenStore) Delete(ctx context.Context, service, key string) error {
	session, err := bs.getSession(ctx)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "bw", "edit", "item", service, "--field", key, "--value", "")
	cmd.Env = append(cmd.Env, "BW_SESSION="+session)
	return cmd.Run()
}

func (bs *BitwardenStore) List(ctx context.Context, service string) (map[string]string, error) {
	session, err := bs.getSession(ctx)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "bw", "get", "item", service)
	cmd.Env = append(cmd.Env, "BW_SESSION="+session)
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
	session, err := bs.getSession(ctx)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "bw", "get", "item", service)
	cmd.Env = append(cmd.Env, "BW_SESSION="+session)
	_, err = cmd.Output()
	return err
}

func (bs *BitwardenStore) Name() string {
	return "bitwarden"
}