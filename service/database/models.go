package database

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Photo    []byte `json:"photo,omitempty"` // stored as BLOB
}

// Conversation represents a chat (group or single)
type Conversation struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"` // Only for groups
	IsGroup bool   `json:"isGroup"`
	Photo   []byte `json:"photo,omitempty"` // Only for groups
	OwnerID string `json:"ownerId"`         // Creator of the group
}

// ConversationUnread adds unread count and last message to Conversation for the list view
type ConversationUnread struct {
	Conversation
	LastMessage *Message `json:"lastMessage"`
	UnreadCount int      `json:"unreadCount"`
}

// ConversationDetails adds members and messages to Conversation for the detail view
type ConversationDetails struct {
	Conversation
	Members  []User    `json:"members"`
	Messages []Message `json:"messages"`
}

// Message represents a text or photo message
type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Content        string     `json:"content"`
	Type           string     `json:"type"` // "text" or "image"
	Timestamp      time.Time  `json:"timestamp"`
	Received       bool       `json:"received"`
	Read           bool       `json:"read"`
	Reactions      []Reaction `json:"reactions"`
	ReplyToID      *string    `json:"replyToId,omitempty"` // Pointer because it can be null
	ReplyTo        *Message   `json:"replyTo,omitempty"`   // Populated by GetConversation
	Photo          []byte     `json:"photo,omitempty"`     // For media messages (text+photo)
}

// Reaction represents a user's reaction to a message
type Reaction struct {
	ID        string `json:"id"`
	MessageID string `json:"messageId"`
	UserID    string `json:"userId"`
	Emoji     string `json:"emoji"`
}
