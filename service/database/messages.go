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

	res, err := db.c.Exec("INSERT INTO messages (conversation_id, sender_id, content, type, timestamp, received, read, reply_to_id, photo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		message.ConversationID, message.SenderID, message.Content, message.Type, message.Timestamp, message.Received, message.Read, message.ReplyToID, message.Photo)
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
	var photo []byte
	err := db.c.QueryRow("SELECT content, type, photo FROM messages WHERE id = ?", forwardedMessageId).Scan(&content, &msgType, &photo)
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
		Content:        "(Forwarded) " + content,
		Type:           msgType,
		Photo:          photo,
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
	// Check if already reacted
	var count int
	err := db.c.QueryRow("SELECT COUNT(*) FROM reactions WHERE message_id = ? AND user_id = ?", reaction.MessageID, reaction.UserID).Scan(&count)
	if err != nil {
		return Reaction{}, err
	}
	if count > 0 {
		return Reaction{}, errors.New("user already reacted to this message")
	}

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
