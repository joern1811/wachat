package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create config file with OpenAI API key",
	Long: `Interactively creates the wachat config file.
Prompts for an OpenAI API key, validates it against the API,
and writes the config to ~/.config/wachat/config.json.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	dir := configDir()
	configPath := filepath.Join(dir, "config.json")

	existingKey := ""

	if _, err := os.Stat(configPath); err == nil {
		existingKey, _ = readExistingKey(configPath)

		fmt.Fprintf(cmd.OutOrStdout(), "Config already exists at %s\n", configPath)
		fmt.Fprint(cmd.OutOrStdout(), "Overwrite? [y/N]: ")

		var answer string
		fmt.Scanln(&answer) //nolint:gosec // interactive CLI input, error not actionable

		if !strings.EqualFold(answer, "y") {
			return nil
		}
	}

	prompt := "OpenAI API Key: "
	if existingKey != "" {
		masked := existingKey[:7] + "***" + existingKey[len(existingKey)-3:]
		prompt = fmt.Sprintf("OpenAI API Key [%s]: ", masked)
	}

	fmt.Fprint(cmd.OutOrStdout(), prompt)

	var apiKey string
	fmt.Scanln(&apiKey) //nolint:gosec // interactive CLI input, error not actionable

	if apiKey == "" && existingKey != "" {
		apiKey = existingKey
	}

	if apiKey == "" {
		return fmt.Errorf("API key must not be empty")
	}

	fmt.Fprint(cmd.OutOrStdout(), "Validating API key... ")

	if err := validateAPIKey(cmd.Context(), apiKey); err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "FAILED")
		return fmt.Errorf("invalid API key: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "OK")

	if err := os.MkdirAll(dir, 0750); err != nil { //nolint:gosec // path from XDG_CONFIG_HOME or user home dir
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(map[string]string{"openai_api_key": apiKey}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Config written to %s\n", configPath)
	return nil
}

func readExistingKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var cfg map[string]string
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}

	return cfg["openai_api_key"], nil
}

func validateAPIKey(ctx context.Context, apiKey string) error {
	client := openai.NewClient(option.WithAPIKey(apiKey))

	_, err := client.Models.List(ctx)
	return err
}
