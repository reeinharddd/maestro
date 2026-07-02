package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/reeinharrrd/maestro/internal/credentials"
	"github.com/spf13/cobra"
)

const systemService = "opencode"

func newCredentialsCmd() *cobra.Command {
	var backend string
	var service string

	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Manage secrets via credential stores (Bitwarden, keyring, file)",
		Long: `Manage API keys, tokens, and passwords through configured credential stores.

Backends:
  bitwarden  (default)  Bitwarden CLI (bw) — secure, syncs across devices
  keyring                OS keyring (requires secret-tool on Linux)
  file                   Encrypted file in config directory

Resolution order when maestro needs a key:
  1. Environment variable (os.Getenv)
  2. Configured credential store (Bitwarden / keyring / file)

Examples:
  maestro credentials set GROQ_API_KEY          # prompt for value
  maestro credentials set GROQ_API_KEY mykey     # inline value
  maestro credentials get GROQ_API_KEY
  maestro credentials list
  maestro credentials delete GROQ_API_KEY
  maestro credentials doctor                     # test store connectivity
  maestro credentials migrate                   # migrate env keys to store
  maestro credentials export-env                 # generate .env from store`,
	}

	cmd.PersistentFlags().StringVar(&backend, "backend", "bitwarden", "Credential store backend (bitwarden, keyring, file)")
	cmd.PersistentFlags().StringVar(&service, "service", systemService, "Service namespace for credentials")

	cmd.AddCommand(newCredentialsSetCmd(&backend, &service))
	cmd.AddCommand(newCredentialsGetCmd(&backend, &service))
	cmd.AddCommand(newCredentialsListCmd(&backend, &service))
	cmd.AddCommand(newCredentialsDeleteCmd(&backend, &service))
	cmd.AddCommand(newCredentialsDoctorCmd(&backend, &service))
	cmd.AddCommand(newCredentialsMigrateCmd(&backend, &service))
	cmd.AddCommand(newCredentialsExportEnvCmd(&backend, &service))

	return cmd
}

func openCredStore(backend, service string) (credentials.CredentialStore, error) {
	return credentials.NewStore(credentials.Config{
		Backend: backend,
		Options: map[string]string{"service": service},
	})
}

func newCredentialsSetCmd(backend, service *string) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> [value]",
		Short: "Store a credential",
		Long:  "Store a credential in the configured store. If value is omitted, reads from stdin.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := ""
			if len(args) >= 2 {
				value = args[1]
			} else {
				fmt.Fprintf(cmd.OutOrStderr(), "Enter value for %s (Ctrl+D to finish):\n", key)
				data, err := readStdin()
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				value = strings.TrimSpace(string(data))
			}

			store, err := openCredStore(*backend, *service)
			if err != nil {
				return fmt.Errorf("open credential store: %w", err)
			}

			if err := store.Set(context.Background(), *service, key, value); err != nil {
				return fmt.Errorf("store %s: %w", key, err)
			}
			fmt.Printf("  OK: %s stored in %s/%s\n", key, *backend, *service)
			return nil
		},
	}
}

func newCredentialsGetCmd(backend, service *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Retrieve a credential (masked)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openCredStore(*backend, *service)
			if err != nil {
				return fmt.Errorf("open credential store: %w", err)
			}

			value, err := store.Get(context.Background(), *service, args[0])
			if err != nil {
				return fmt.Errorf("get %s: %w", args[0], err)
			}
			if value == "" {
				return fmt.Errorf("%s: not found in %s/%s", args[0], *backend, *service)
			}

			fmt.Printf("%s=%s\n", args[0], maskKey(value))
			return nil
		},
	}
}

func newCredentialsListCmd(backend, service *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored credentials (masked)",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openCredStore(*backend, *service)
			if err != nil {
				return fmt.Errorf("open credential store: %w", err)
			}

			creds, err := store.List(context.Background(), *service)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			if len(creds) == 0 {
				fmt.Println("  No credentials found.")
				return nil
			}

			keys := make([]string, 0, len(creds))
			for k := range creds {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Printf("  %s = %s\n", k, maskKey(creds[k]))
			}
			return nil
		},
	}
}

func newCredentialsDeleteCmd(backend, service *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openCredStore(*backend, *service)
			if err != nil {
				return fmt.Errorf("open credential store: %w", err)
			}

			if err := store.Delete(context.Background(), *service, args[0]); err != nil {
				return fmt.Errorf("delete %s: %w", args[0], err)
			}
			fmt.Printf("  OK: %s removed from %s/%s\n", args[0], *backend, *service)
			return nil
		},
	}
}

func newCredentialsDoctorCmd(backend, service *string) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Test credential store connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openCredStore(*backend, *service)
			if err != nil {
				return fmt.Errorf("open credential store: %w", err)
			}

			fmt.Printf("Backend: %s (%s)\n", store.Name(), *backend)
			fmt.Printf("Service: %s\n", *service)

			if err := store.Test(context.Background(), *service); err != nil {
				fmt.Printf("  FAIL: %v\n", err)
				return fmt.Errorf("store test: %w", err)
			}
			return nil
		},
	}
}

// knownEnvKeys lists the environment variables that opencode providers commonly reference.
var knownEnvKeys = []string{
	"GROQ_API_KEY",
	"MISTRAL_API_KEY",
	"NVIDIA_API_KEY",
	"OPENROUTER_API_KEY",
	"CEREBRAS_API_KEY",
	"GITHUB_TOKEN",
	"OPENCODE_ZEN_API_KEY",
	"OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"DEEPSEEK_API_KEY",
	"GOOGLE_API_KEY",
	"COHERE_API_KEY",
	"AI21_API_KEY",
	"REPLICATE_API_TOKEN",
	"TOGETHER_API_KEY",
	"FIREWORKS_API_KEY",
	"PERPLEXITY_API_KEY",
}

func newCredentialsMigrateCmd(backend, service *string) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate env vars to credential store",
		Long: `Scan environment variables for known API keys and store any that
are not yet in the credential store.

Safe to re-run: skips keys already stored, adds only new ones.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openCredStore(*backend, *service)
			if err != nil {
				return fmt.Errorf("open credential store: %w", err)
			}

			existing, err := store.List(context.Background(), *service)
			if err != nil {
				existing = map[string]string{}
			}

			var migrated, skipped int
			for _, key := range knownEnvKeys {
				if _, exists := existing[key]; exists {
					skipped++
					continue
				}
				val := os.Getenv(key)
				if val == "" {
					continue
				}
				if err := store.Set(context.Background(), *service, key, val); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "  WARN: %s: %v\n", key, err)
					continue
				}
				migrated++
				fmt.Printf("  OK: %s migrated\n", key)
			}
			fmt.Printf("\nDone: %d migrated, %d skipped (already stored), %d not in env\n",
				migrated, skipped, len(knownEnvKeys)-migrated-skipped)
			return nil
		},
	}
}

func newCredentialsExportEnvCmd(backend, service *string) *cobra.Command {
	return &cobra.Command{
		Use:   "export-env",
		Short: "Export credentials as .env format",
		Long: `Print all stored credentials in KEY=VALUE format, safe to redirect
into opencode.env or .env files.

  maestro credentials export-env > ~/.config/opencode/opencode.env`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openCredStore(*backend, *service)
			if err != nil {
				return fmt.Errorf("open credential store: %w", err)
			}

			creds, err := store.List(context.Background(), *service)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			keys := make([]string, 0, len(creds))
			for k := range creds {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", k, creds[k])
			}
			return nil
		},
	}
}

func readStdin() ([]byte, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return os.ReadFile("/dev/stdin")
	}
	data := make([]byte, 4096)
	n, err := os.Stdin.Read(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}
