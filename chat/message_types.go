package chat

import "time"

type AttachmentType string
type ReactionType string

const (
	ImageAttachment    AttachmentType = "image"
	DocumentAttachment AttachmentType = "document"
	FileAttachment     AttachmentType = "file"

	EmojiReaction ReactionType = "emoji"
	RaiseHand     ReactionType = "raise_hand"
)

type Attachment struct {
	Type        AttachmentType `json:"type"`
	URL         string         `json:"url"`
	Name        string         `json:"name"`
	Size        int64          `json:"size"`
	ContentType string         `json:"contentType"`
}

type Reaction struct {
	Type      ReactionType `json:"type"`
	Content   string       `json:"content"`
	UserID    string       `json:"userId"`
	Timestamp time.Time    `json:"timestamp"`
}
