package api

import "github.com/kubilaykoccc/Wasa/service/database"

// structures.go contains the Go structs for API requests and responses that differ from the DB models

// LoginRequest is the body for POST /session
type LoginRequest struct {
	Name string `json:"name"`
}

// LoginResponse is the response for POST /session
type LoginResponse struct {
	Identifier uint64 `json:"identifier"`
}

// UserResponse is the response for user endpoints (often aliases database.User but specific for API)
type UserResponse struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Photo    []byte `json:"photo,omitempty"`
}

func FromDatabaseUser(u database.User) UserResponse {
	return UserResponse{
		ID:       u.ID,
		Username: u.Username,
		Photo:    u.Photo,
	}
}

// ConversationUnreadResponse for list view
type ConversationUnreadResponse struct {
	ID          uint64           `json:"id"`
	Name        string           `json:"name,omitempty"`
	IsGroup     bool             `json:"isGroup"`
	Photo       []byte           `json:"photo,omitempty"`
	LastMessage *MessageResponse `json:"lastMessage"`
	UnreadCount int              `json:"unreadCount"`
}

func FromDatabaseConversationUnread(c database.ConversationUnread) ConversationUnreadResponse {
	var lm *MessageResponse
	if c.LastMessage != nil {
		m := FromDatabaseMessage(*c.LastMessage)
		lm = &m
	}
	return ConversationUnreadResponse{
		ID:          c.ID,
		Name:        c.Name,
		IsGroup:     c.IsGroup,
		Photo:       c.Photo,
		LastMessage: lm,
		UnreadCount: c.UnreadCount,
	}
}

// MessageResponse
type MessageResponse struct {
	ID             uint64             `json:"id"`
	ConversationID uint64             `json:"conversationId"`
	SenderID       uint64             `json:"senderId"`
	Content        string             `json:"content"`
	Type           string             `json:"type"`
	Timestamp      string             `json:"timestamp"` // ISO8601 string
	Received       bool               `json:"received"`
	Read           bool               `json:"read"`
	Reactions      []ReactionResponse `json:"reactions"`
	ReplyTo        *MessageResponse   `json:"replyTo,omitempty"`
	Photo          []byte             `json:"photo,omitempty"`
}

func FromDatabaseMessage(m database.Message) MessageResponse {
	reactions := make([]ReactionResponse, len(m.Reactions))
	for i, r := range m.Reactions {
		reactions[i] = FromDatabaseReaction(r)
	}
	msg := MessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Content:        m.Content,
		Type:           m.Type,
		Timestamp:      m.Timestamp.Format("2006-01-02T15:04:05.999Z07:00"),
		Received:       m.Received,
		Read:           m.Read,
		Reactions:      reactions,
		Photo:          m.Photo,
	}

	if m.ReplyTo != nil {
		reply := FromDatabaseMessage(*m.ReplyTo)
		msg.ReplyTo = &reply
	}
	return msg
}

// ReactionResponse
type ReactionResponse struct {
	ID        uint64 `json:"id"`
	MessageID uint64 `json:"messageId"`
	UserID    uint64 `json:"userId"`
	Emoji     string `json:"emoji"`
}

func FromDatabaseReaction(r database.Reaction) ReactionResponse {
	return ReactionResponse{
		ID:        r.ID,
		MessageID: r.MessageID,
		UserID:    r.UserID,
		Emoji:     r.Emoji,
	}
}

// SetUsernameRequest
type SetUsernameRequest struct {
	Name string `json:"name"`
}

// SetGroupNameRequest
type SetGroupNameRequest struct {
	Name string `json:"name"`
}

// SendMessageRequest
type SendMessageRequest struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Text      string `json:"text,omitempty"`
	ReplyToID uint64 `json:"replyToId,omitempty"`
}

// ForwardMessageRequest
type ForwardMessageRequest struct {
	ForwardedMessageID uint64 `json:"forwardedMessageId"`
}

// CommentMessageRequest
type CommentMessageRequest struct {
	Emoji string `json:"emoji"`
}

// CreateConversationRequest
type CreateConversationRequest struct {
	Name    string   `json:"name"`
	Members []uint64 `json:"members"`
}
