package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/joern1811/wachat/internal/domain"
)

// ChatService orchestrates the chat processing pipeline.
type ChatService struct {
	parser      domain.ChatParser
	transcriber domain.Transcriber
	renderer    domain.ChatRenderer
}

func NewChatService(parser domain.ChatParser, transcriber domain.Transcriber, renderer domain.ChatRenderer) *ChatService {
	return &ChatService{
		parser:      parser,
		transcriber: transcriber,
		renderer:    renderer,
	}
}

// Process runs the full pipeline: parse → filter → transcribe → render.
func (s *ChatService) Process(ctx context.Context, exportPath string, from, to *time.Time, w io.Writer) error {
	chat, err := s.parser.Parse(exportPath)
	if err != nil {
		return fmt.Errorf("parsing export: %w", err)
	}

	// Apply time filter before transcription to avoid unnecessary API calls
	if from != nil || to != nil {
		chat = chat.Filter(from, to)
	}

	// Transcribe voice messages
	for i := range chat.Messages {
		if chat.Messages[i].Type != domain.VoiceMessage {
			continue
		}

		text, err := s.transcriber.Transcribe(ctx, chat.Messages[i].MediaRef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: transcription failed for %s: %v\n", chat.Messages[i].MediaRef, err)
			continue
		}
		chat.Messages[i].Content = text
	}

	return s.renderer.Render(w, chat)
}
