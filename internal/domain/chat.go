package domain

import "time"

type Chat struct {
	Messages []Message
}

// Filter returns a new Chat containing only messages within the given time range.
// nil values for from/to mean no lower/upper bound.
func (c *Chat) Filter(from, to *time.Time) *Chat {
	filtered := &Chat{}
	for _, msg := range c.Messages {
		if from != nil && msg.Timestamp.Before(*from) {
			continue
		}
		if to != nil && msg.Timestamp.After(*to) {
			continue
		}
		filtered.Messages = append(filtered.Messages, msg)
	}
	return filtered
}
