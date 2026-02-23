package parser

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/joern1811/wachat/internal/domain"
)

// WhatsAppParser parses WhatsApp chat exports (.zip files).
type WhatsAppParser struct {
	// TempDir holds the path to the extracted files (set after Parse).
	TempDir string
}

// Regex patterns for WhatsApp message lines.
// Supports:
//   - iOS DE:     [DD.MM.YY, HH:MM:SS] Sender: Text
//   - iOS DE alt: [DD.MM.YYYY, HH:MM:SS] Sender: Text
//   - Android DE: DD.MM.YY, HH:MM - Sender: Text
var (
	// iOS format: [DD.MM.YY(YY), HH:MM:SS] Sender: Text
	iosRe = regexp.MustCompile(`^\[(\d{2}\.\d{2}\.\d{2,4}), (\d{2}:\d{2}:\d{2})\] ([^:]+): (.*)$`)
	// Android DE format: DD.MM.YY, HH:MM - Sender: Text
	androidRe = regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{2}), (\d{2}:\d{2}) - ([^:]+): (.*)$`)
	// System messages (no sender)
	iosSystemRe     = regexp.MustCompile(`^\[(\d{2}\.\d{2}\.\d{2,4}), (\d{2}:\d{2}:\d{2})\] (.*)$`)
	androidSystemRe = regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{2}), (\d{2}:\d{2}) - (.*)$`)

	// Attachment patterns
	attachedRe = regexp.MustCompile(`\s*\(Datei angehängt\)\s*$|\s*\(file attached\)\s*$`)
	// <Anhang: filename> or <attached: filename>
	anhangRe = regexp.MustCompile(`^<(?:Anhang|attached|Media omitted): (.+)>$`)
)

func (p *WhatsAppParser) Parse(exportPath string) (*domain.Chat, error) {
	tempDir, err := os.MkdirTemp("", "wachat-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	p.TempDir = tempDir

	if err := extractZip(exportPath, tempDir); err != nil {
		return nil, fmt.Errorf("extracting zip: %w", err)
	}

	txtFile, err := findChatFile(tempDir)
	if err != nil {
		return nil, fmt.Errorf("finding chat file: %w", err)
	}

	messages, err := parseTextFile(txtFile)
	if err != nil {
		return nil, fmt.Errorf("parsing chat file: %w", err)
	}

	// Set full paths for media references
	for i := range messages {
		if messages[i].MediaRef != "" {
			messages[i].MediaRef = filepath.Join(tempDir, messages[i].MediaRef)
		}
	}

	return &domain.Chat{Messages: messages}, nil
}

// Cleanup removes the temporary directory.
func (p *WhatsAppParser) Cleanup() {
	if p.TempDir != "" {
		_ = os.RemoveAll(p.TempDir)
	}
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Sanitize path to prevent zip slip (G305)
		name := filepath.Clean(f.Name)
		if strings.Contains(name, "..") {
			continue
		}
		destPath := filepath.Join(destDir, name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o750); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
			return err
		}

		if err := extractZipFile(f, destPath); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, destPath string) error {
	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Limit extraction size to 1 GB to prevent decompression bombs (G110)
	const maxSize = 1 << 30
	_, err = io.Copy(outFile, io.LimitReader(rc, maxSize))
	return err
}

func findChatFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no .txt chat file found in export")
}

// stripInvisible removes Unicode control characters (LTR mark, zero-width spaces, etc.)
func stripInvisible(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\u200e' || r == '\u200f': // LTR / RTL mark
			return -1
		case r == '\u200b' || r == '\u200c' || r == '\u200d': // zero-width spaces
			return -1
		case r == '\ufeff': // BOM
			return -1
		default:
			return r
		}
	}, s)
}

func parseTextFile(path string) ([]domain.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var messages []domain.Message
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := stripInvisible(scanner.Text())

		if msg, ok := parseMessageLine(line); ok {
			messages = append(messages, msg)
		} else if len(messages) > 0 {
			// Multiline: append to previous message content
			messages[len(messages)-1].Content += "\n" + line
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func parseMessageLine(line string) (domain.Message, bool) {
	// Try iOS format first: [DD.MM.YY(YY), HH:MM:SS] Sender: Text
	if m := iosRe.FindStringSubmatch(line); m != nil {
		ts, err := parseIOSTimestamp(m[1], m[2])
		if err != nil {
			return domain.Message{}, false
		}
		content, msgType, mediaRef := classifyContent(m[4])
		return domain.Message{
			Timestamp: ts,
			Sender:    m[3],
			Content:   content,
			Type:      msgType,
			MediaRef:  mediaRef,
		}, true
	}

	// Try Android format: DD.MM.YY, HH:MM - Sender: Text
	if m := androidRe.FindStringSubmatch(line); m != nil {
		ts, err := time.Parse("02.01.06, 15:04", m[1]+", "+m[2])
		if err != nil {
			return domain.Message{}, false
		}
		content, msgType, mediaRef := classifyContent(m[4])
		return domain.Message{
			Timestamp: ts,
			Sender:    m[3],
			Content:   content,
			Type:      msgType,
			MediaRef:  mediaRef,
		}, true
	}

	// Try iOS system message
	if m := iosSystemRe.FindStringSubmatch(line); m != nil {
		ts, err := parseIOSTimestamp(m[1], m[2])
		if err != nil {
			return domain.Message{}, false
		}
		return domain.Message{
			Timestamp: ts,
			Content:   m[3],
			Type:      domain.SystemMessage,
		}, true
	}

	// Try Android system message
	if m := androidSystemRe.FindStringSubmatch(line); m != nil {
		ts, err := time.Parse("02.01.06, 15:04", m[1]+", "+m[2])
		if err != nil {
			return domain.Message{}, false
		}
		return domain.Message{
			Timestamp: ts,
			Content:   m[3],
			Type:      domain.SystemMessage,
		}, true
	}

	return domain.Message{}, false
}

func parseIOSTimestamp(datePart, timePart string) (time.Time, error) {
	// Try 2-digit year first (DD.MM.YY), then 4-digit (DD.MM.YYYY)
	if ts, err := time.Parse("02.01.06, 15:04:05", datePart+", "+timePart); err == nil {
		return ts, nil
	}
	return time.Parse("02.01.2006, 15:04:05", datePart+", "+timePart)
}

func classifyContent(content string) (string, domain.MessageType, string) {
	// Check for <Anhang: filename> pattern
	if m := anhangRe.FindStringSubmatch(content); m != nil {
		filename := m[1]
		return classifyMediaFile(filename)
	}

	// Check for (Datei angehängt) / (file attached) suffix
	cleaned := attachedRe.ReplaceAllString(content, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned != strings.TrimSpace(content) {
		return classifyMediaFile(cleaned)
	}

	return content, domain.TextMessage, ""
}

func classifyMediaFile(filename string) (string, domain.MessageType, string) {
	if isVoiceMessage(filename) {
		return filename, domain.VoiceMessage, filename
	}
	if isImageMessage(filename) {
		return filename, domain.ImageMessage, filename
	}
	if isVideoMessage(filename) {
		return filename, domain.VideoMessage, filename
	}
	return filename, domain.DocumentMessage, filename
}

func isVoiceMessage(s string) bool {
	upper := strings.ToUpper(s)
	lower := strings.ToLower(s)
	return strings.HasPrefix(upper, "PTT-") ||
		strings.HasPrefix(upper, "AUD-") ||
		strings.Contains(upper, "-AUDIO-") ||
		strings.HasSuffix(lower, ".opus") ||
		strings.HasSuffix(lower, ".m4a") ||
		strings.HasSuffix(lower, ".ogg")
}

func isImageMessage(s string) bool {
	upper := strings.ToUpper(s)
	return strings.HasPrefix(upper, "IMG-") ||
		strings.HasPrefix(upper, "PHOTO-") ||
		strings.Contains(upper, "-PHOTO-")
}

func isVideoMessage(s string) bool {
	upper := strings.ToUpper(s)
	return strings.HasPrefix(upper, "VID-") ||
		strings.HasPrefix(upper, "VIDEO-") ||
		strings.Contains(upper, "-VIDEO-")
}
