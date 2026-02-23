package domain

import (
	"context"
	"io"
)

// ChatParser parses a WhatsApp export into a Chat.
type ChatParser interface {
	Parse(exportPath string) (*Chat, error)
}

// Transcriber transcribes an audio file to text.
type Transcriber interface {
	Transcribe(ctx context.Context, audioPath string) (string, error)
}

// ChatRenderer renders a Chat to an output writer.
type ChatRenderer interface {
	Render(w io.Writer, chat *Chat) error
}
