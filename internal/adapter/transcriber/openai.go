package transcriber

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go/v3"
)

// OpenAITranscriber transcribes audio files using the OpenAI Whisper API.
// The API key is read from the OPENAI_API_KEY environment variable by the SDK.
type OpenAITranscriber struct {
	client openai.Client
}

func NewOpenAITranscriber() *OpenAITranscriber {
	return &OpenAITranscriber{
		client: openai.NewClient(),
	}
}

func (t *OpenAITranscriber) Transcribe(ctx context.Context, audioPath string) (string, error) {
	// Whisper doesn't accept .opus directly, but WhatsApp .opus files are
	// actually OGG/Opus containers. Symlink with .ogg extension so the API
	// accepts the file.
	actualPath := audioPath
	if strings.ToLower(filepath.Ext(audioPath)) == ".opus" {
		oggPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".ogg"
		if err := os.Symlink(audioPath, oggPath); err == nil {
			actualPath = oggPath
			defer os.Remove(oggPath)
		}
	}

	f, err := os.Open(actualPath)
	if err != nil {
		return "", fmt.Errorf("opening audio file %s: %w", actualPath, err)
	}
	defer f.Close()

	transcription, err := t.client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		Model: openai.AudioModelWhisper1,
		File:  f,
	})
	if err != nil {
		return "", fmt.Errorf("transcribing %s: %w", audioPath, err)
	}

	return transcription.Text, nil
}
