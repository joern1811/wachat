package renderer

import (
	"fmt"
	"io"

	"github.com/joern1811/wachat/internal/domain"
)

// TextRenderer renders a chat as plain text.
type TextRenderer struct {
	Markdown bool
}

func (r *TextRenderer) Render(w io.Writer, chat *domain.Chat) error {
	for i := range chat.Messages {
		line := r.formatMessage(&chat.Messages[i])
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func (r *TextRenderer) formatMessage(msg *domain.Message) string {
	ts := msg.Timestamp.Format("02.01.2006 15:04")

	switch msg.Type {
	case domain.SystemMessage:
		if r.Markdown {
			return fmt.Sprintf("*[%s] %s*", ts, msg.Content)
		}
		return fmt.Sprintf("*** [%s] %s", ts, msg.Content)

	case domain.VoiceMessage:
		prefix := "[Sprachnachricht]"
		content := msg.Content
		if content == "" || content == msg.MediaRef {
			content = msg.MediaRef
		}
		return fmt.Sprintf("[%s] %s: %s %s", ts, msg.Sender, prefix, content)

	case domain.ImageMessage:
		return fmt.Sprintf("[%s] %s: [Bild] %s", ts, msg.Sender, msg.MediaRef)

	case domain.VideoMessage:
		return fmt.Sprintf("[%s] %s: [Video] %s", ts, msg.Sender, msg.MediaRef)

	case domain.DocumentMessage:
		return fmt.Sprintf("[%s] %s: [Dokument] %s", ts, msg.Sender, msg.MediaRef)

	default:
		return fmt.Sprintf("[%s] %s: %s", ts, msg.Sender, msg.Content)
	}
}
