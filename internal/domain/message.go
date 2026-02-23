package domain

import "time"

type MessageType int

const (
	TextMessage MessageType = iota
	VoiceMessage
	ImageMessage
	VideoMessage
	DocumentMessage
	SystemMessage
)

type Message struct {
	Timestamp time.Time
	Sender    string
	Content   string      // Text or transcribed text
	Type      MessageType
	MediaRef  string      // Filename (e.g. "PTT-20240115-WA0000.opus")
}
