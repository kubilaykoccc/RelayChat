package database

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Photo    []byte `json:"photo,omitempty"` // stored as BLOB
}

// Conversation represents a chat (group or single)
type Conversation struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name,omitempty"` // Only for groups
	IsGroup bool   `json:"isGroup"`
	Photo   []byte `json:"photo,omitempty"` // Only for groups
	OwnerID uint64 `json:"ownerId"`         // Creator of the group
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
	ID             uint64     `json:"id"`
	ConversationID uint64     `json:"conversationId"`
	SenderID       uint64     `json:"senderId"`
	Content        string     `json:"content"`
	Type           string     `json:"type"` // "text" or "image"
	Timestamp      time.Time  `json:"timestamp"`
	Received       bool       `json:"received"`
	Read           bool       `json:"read"`
	Reactions      []Reaction `json:"reactions"`
}

// Reaction represents a user's reaction to a message
type Reaction struct {
	ID        uint64 `json:"id"`
	MessageID uint64 `json:"messageId"`
	UserID    uint64 `json:"userId"`
	Emoji     string `json:"emoji"`
}
