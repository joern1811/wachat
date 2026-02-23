package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/joern1811/wachat/internal/adapter/parser"
	"github.com/joern1811/wachat/internal/adapter/renderer"
	"github.com/joern1811/wachat/internal/adapter/transcriber"
	"github.com/joern1811/wachat/internal/app"
)

var (
	fromStr string
	toStr   string
	output  string
	format  string
)

var rootCmd = &cobra.Command{
	Use:   "wachat <export.zip>",
	Short: "Convert WhatsApp chat exports to readable text",
	Long: `wachat processes WhatsApp chat exports (.zip files) and converts them
to readable text or markdown. Voice messages are automatically transcribed
using the OpenAI Whisper API.`,
	Args: cobra.ExactArgs(1),
	RunE: runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVar(&fromStr, "from", "", `Start time filter (format: "DD.MM.YYYY" or "DD.MM.YYYY HH:MM")`)
	rootCmd.Flags().StringVar(&toStr, "to", "", `End time filter (format: "DD.MM.YYYY" or "DD.MM.YYYY HH:MM")`)
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	rootCmd.Flags().StringVarP(&format, "format", "f", "text", `Output format: "text" or "markdown"`)
}

func configDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		configHome = filepath.Join(home, ".config")
	}

	return filepath.Clean(filepath.Join(configHome, app.ApplicationName))
}

func initConfig() {
	dir := configDir()

	if _, err := os.Stat(dir); os.IsNotExist(err) { //nolint:gosec // path is constructed from XDG_CONFIG_HOME or user home dir
		err = os.MkdirAll(dir, 0750) //nolint:gosec // see above
		cobra.CheckErr(err)
	}

	viper.AddConfigPath(dir)
	viper.SetConfigType("json")
	viper.SetConfigName("config")

	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()

	// Silently ignore missing config file
	_ = viper.ReadInConfig()

	// Bridge config value to environment variable for OpenAI SDK
	if apiKey := viper.GetString("openai_api_key"); apiKey != "" {
		if os.Getenv("OPENAI_API_KEY") == "" {
			_ = os.Setenv("OPENAI_API_KEY", apiKey)
		}
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	exportPath := args[0]

	from, err := parseTime(fromStr)
	if err != nil {
		return fmt.Errorf("parsing --from: %w", err)
	}

	to, err := parseTime(toStr)
	if err != nil {
		return fmt.Errorf("parsing --to: %w", err)
	}

	// If --to is date-only, set to end of day
	if to != nil && !strings.Contains(toStr, " ") {
		endOfDay := to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		to = &endOfDay
	}

	p := &parser.WhatsAppParser{}
	t := transcriber.NewOpenAITranscriber()
	r := &renderer.TextRenderer{Markdown: format == "markdown"}

	svc := app.NewChatService(p, t, r)

	w := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	ctx := context.Background()
	if err := svc.Process(ctx, exportPath, from, to, w); err != nil {
		p.Cleanup()
		return err
	}

	p.Cleanup()
	return nil
}

func parseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}

	formats := []string{
		"02.01.2006 15:04",
		"02.01.2006",
	}

	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unknown time format: %q (expected DD.MM.YYYY or DD.MM.YYYY HH:MM)", s)
}
