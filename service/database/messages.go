package database

import (
	"database/sql"
	"errors"
	"time"
)

// SendMessage adds a new message to the database
func (db *appdbimpl) SendMessage(message Message) (Message, error) {
	message.Timestamp = time.Now()
	message.Received = false
	message.Read = false

	res, err := db.c.Exec("INSERT INTO messages (conversation_id, sender_id, content, type, timestamp, received, read) VALUES (?, ?, ?, ?, ?, ?, ?)",
		message.ConversationID, message.SenderID, message.Content, message.Type, message.Timestamp, message.Received, message.Read)
	if err != nil {
		return Message{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Message{}, err
	}
	message.ID = uint64(id)
	return message, nil
}

// ForwardMessage creates a new message with content from another message
func (db *appdbimpl) ForwardMessage(conversationId uint64, forwardedMessageId uint64, senderId uint64) (Message, error) {
	// Get original content
	var content, msgType string
	err := db.c.QueryRow("SELECT content, type FROM messages WHERE id = ?", forwardedMessageId).Scan(&content, &msgType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Message{}, errors.New("forwarded message not found")
		}
		return Message{}, err
	}

	// Create new message
	newMessage := Message{
		ConversationID: conversationId,
		SenderID:       senderId,
		Content:        content,
		Type:           msgType,
	}
	return db.SendMessage(newMessage)
}

// DeleteMessage deletes a message if the user is the sender
func (db *appdbimpl) DeleteMessage(messageId uint64, userId uint64) error {
	res, err := db.c.Exec("DELETE FROM messages WHERE id = ? AND sender_id = ?", messageId, userId)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("message not found or not owned by user")
	}
	return nil
}

// CommentMessage adds a reaction
func (db *appdbimpl) CommentMessage(reaction Reaction) (Reaction, error) {
	res, err := db.c.Exec("INSERT INTO reactions (message_id, user_id, emoji) VALUES (?, ?, ?)",
		reaction.MessageID, reaction.UserID, reaction.Emoji)
	if err != nil {
		return Reaction{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Reaction{}, err
	}
	reaction.ID = uint64(id)
	return reaction, nil
}

// UncommentMessage removes a reaction
func (db *appdbimpl) UncommentMessage(reactionId uint64, userId uint64) error {
	res, err := db.c.Exec("DELETE FROM reactions WHERE id = ? AND user_id = ?", reactionId, userId)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("reaction not found or not owned by user")
	}
	return nil
}
