package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreateConversation creates a new conversation (could be direct or group, starting as direct usually)
func (db *appdbimpl) CreateConversation(ownerId uint64, otherUserId uint64) (Conversation, error) {
	// SQL to find common conversation (is_group=0)
	query := `
		SELECT c.id
		FROM conversations c
		JOIN conversation_members cm1 ON c.id = cm1.conversation_id
		JOIN conversation_members cm2 ON c.id = cm2.conversation_id
		WHERE c.is_group = 0 AND cm1.user_id = ? AND cm2.user_id = ?
	`
	var existingId uint64
	err := db.c.QueryRow(query, ownerId, otherUserId).Scan(&existingId)
	if err == nil {
		// Found existing
		return Conversation{ID: existingId, IsGroup: false}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, fmt.Errorf("error checking existing conversation: %w", err)
	}

	// Create new transaction
	tx, err := db.c.Begin()
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback()

	// Insert Conversation
	res, err := tx.Exec("INSERT INTO conversations (is_group, owner_id) VALUES (0, ?)", ownerId)
	if err != nil {
		return Conversation{}, err
	}
	cID, err := res.LastInsertId()
	if err != nil {
		return Conversation{}, err
	}
	convoId := uint64(cID)

	// Insert Members
	now := time.Now()
	_, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id, last_seen) VALUES (?, ?, ?)", convoId, ownerId, now)
	if err != nil {
		return Conversation{}, err
	}
	_, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id, last_seen) VALUES (?, ?, ?)", convoId, otherUserId, now)
	if err != nil {
		return Conversation{}, err
	}

	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}

	return Conversation{ID: convoId, IsGroup: false, OwnerID: ownerId}, nil
}

// CreateGroup creates a new group conversation
func (db *appdbimpl) CreateGroup(name string, ownerId uint64) (Conversation, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback()

	// Insert Conversation
	res, err := tx.Exec("INSERT INTO conversations (name, is_group, owner_id) VALUES (?, 1, ?)", name, ownerId)
	if err != nil {
		return Conversation{}, err
	}
	cID, err := res.LastInsertId()
	if err != nil {
		return Conversation{}, err
	}
	convoId := uint64(cID)

	// Add owner as member
	_, err = tx.Exec("INSERT INTO conversation_members (conversation_id, user_id, last_seen) VALUES (?, ?, ?)", convoId, ownerId, time.Now())
	if err != nil {
		return Conversation{}, err
	}

	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}

	return Conversation{ID: convoId, Name: name, IsGroup: true, OwnerID: ownerId}, nil
}

// GetMyConversations returns the list of conversations for the user
func (db *appdbimpl) GetMyConversations(userId uint64) ([]ConversationUnread, error) {
	// Update last_delivered for the user in all conversations they are part of?
	// Actually, strictly speaking, "received in their conversation list" means we fetched the list.
	// We need to update last_delivered for ALL conversations returned.
	// But we don't know the IDs yet.
	// So we fetch, then update? Or update broadly?
	// Broad update on membership:
	_, _ = db.c.Exec("UPDATE conversation_members SET last_delivered = ? WHERE user_id = ?", time.Now(), userId)

	// Trigger status update for messages in these conversations
	// We can do this efficiently by looking for pending messages.
	// For every message that is not received (received=0) AND sender != userId:
	// Check if ALL members of that conv have last_delivered >= msg.timestamp.
	// Because checking ONE BY ONE is slow, we might use a complex query or just do it for active conversations.
	// Given the scale, let's try a query updates approach.

	// SQLite query to update 'received'
	updateReceived := `
        UPDATE messages
        SET received = 1
        WHERE received = 0 
        AND sender_id != ?
        AND (
            SELECT COUNT(*) 
            FROM conversation_members cm 
            WHERE cm.conversation_id = messages.conversation_id 
            AND cm.user_id != messages.sender_id
            AND cm.last_delivered < messages.timestamp
        ) = 0
    `
	// Explanation: Set received=1 if there are NO members (recipients) who have last_delivered < msg.timestamp.
	// i.e. ALL recipients have last_delivered >= msg.timestamp.
	// Using '<' catches anyone who hasn't delivered yet.

	_, err := db.c.Exec(updateReceived, userId)
	if err != nil {
		// Log error but continue?
		fmt.Println("Error updating received status:", err)
	}

	// Query to fetch conversations with dynamic naming for 1-to-1
	// We join with conversation_members again (cm2) to find the OTHER user in 1-to-1 chats (where is_group=0)
	// Then we join with users (u) to get their username and photo.
	// If is_group=1, we use c.name and c.photo.
	// If is_group=0, we use u.username and u.photo.
	query := `
		SELECT 
            c.id, 
            CASE WHEN c.is_group = 1 THEN c.name ELSE u.username END,
            c.is_group, 
            CASE WHEN c.is_group = 1 THEN c.photo ELSE u.photo END,
            c.owner_id, 
            cm.last_seen
		FROM conversations c
		JOIN conversation_members cm ON c.id = cm.conversation_id
        LEFT JOIN conversation_members cm2 ON c.id = cm2.conversation_id AND cm2.user_id != ? AND c.is_group = 0
        LEFT JOIN users u ON cm2.user_id = u.id
		WHERE cm.user_id = ?
	`
	// We pass userId twice: first for cm2 check (!= userId), second for main WHERE clause (cm.user_id = userId)
	rows, err := db.c.Query(query, userId, userId)
	if err != nil {
		return nil, fmt.Errorf("error getting conversations: %w", err)
	}
	defer rows.Close()

	var results []ConversationUnread
	for rows.Next() {
		var c ConversationUnread
		var lastSeen time.Time
		var name sql.NullString
		var photo []byte
		var ownerId sql.NullInt64

		if err := rows.Scan(&c.ID, &name, &c.IsGroup, &photo, &ownerId, &lastSeen); err != nil {
			return nil, fmt.Errorf("error scanning conversation: %w", err)
		}
		c.Name = name.String
		c.Photo = photo
		if ownerId.Valid {
			c.OwnerID = uint64(ownerId.Int64)
		}

		// Get Last Message
		var m Message
		var mTimestamp time.Time
		msgQuery := `
			SELECT id, conversation_id, sender_id, content, type, timestamp, received, read
			FROM messages
			WHERE conversation_id = ?
			ORDER BY timestamp DESC
			LIMIT 1
		`
		err = db.c.QueryRow(msgQuery, c.ID).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &mTimestamp, &m.Received, &m.Read)
		if err == nil {
			m.Timestamp = mTimestamp
			c.LastMessage = &m
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("error getting last message: %w", err)
		}

		// Calculate unread count (messages not read by ME)
		countQuery := `SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND sender_id != ? AND read = 0`
		err = db.c.QueryRow(countQuery, c.ID, userId).Scan(&c.UnreadCount)
		if err != nil {
			return nil, fmt.Errorf("error counting unread: %w", err)
		}

		results = append(results, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// GetConversation returns details (messages and members)
func (db *appdbimpl) GetConversation(conversationId uint64, userId uint64) (ConversationDetails, error) {
	// Verify membership
	var membershipCount int
	err := db.c.QueryRow("SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?", conversationId, userId).Scan(&membershipCount)
	if err != nil {
		return ConversationDetails{}, err
	}
	if membershipCount == 0 {
		return ConversationDetails{}, errors.New("access denied or conversation not found")
	}

	// Update last_seen
	_, _ = db.c.Exec("UPDATE conversation_members SET last_seen = ? WHERE conversation_id = ? AND user_id = ?", time.Now(), conversationId, userId)

	// Update last_delivered as well, since opening a conversation implies delivery
	_, _ = db.c.Exec("UPDATE conversation_members SET last_delivered = ? WHERE conversation_id = ? AND user_id = ?", time.Now(), conversationId, userId)

	// Strict Read Logic:
	// Update 'read' to 1 Only if ALL recipients have last_seen >= msg.timestamp.
	updateRead := `
        UPDATE messages
        SET read = 1
        WHERE conversation_id = ?
        AND read = 0
        AND sender_id != ?
        AND (
            SELECT COUNT(*)
            FROM conversation_members cm
            WHERE cm.conversation_id = messages.conversation_id
            AND cm.user_id != messages.sender_id
            AND cm.last_seen < messages.timestamp
        ) = 0
    `
	// Also trigger updateReceived for this conversation to be safe
	updateReceived := `
        UPDATE messages
        SET received = 1
        WHERE conversation_id = ?
        AND received = 0
        AND sender_id != ?
        AND (
            SELECT COUNT(*) 
            FROM conversation_members cm 
            WHERE cm.conversation_id = messages.conversation_id 
            AND cm.user_id != messages.sender_id
            AND cm.last_delivered < messages.timestamp
        ) = 0
    `
	_, _ = db.c.Exec(updateReceived, conversationId, userId)
	_, _ = db.c.Exec(updateRead, conversationId, userId)

	// Fetch details
	// Similar logic: if not group, we need to fetch the other user's name/photo.
	// However, since we fetch Members later, it might be easier to fetch basic info first, then if 1-to-1, override from members list.
	// BUT, let's do it in SQL for consistency and speed if possible, or just post-process.
	// Post-processing is safer here because we fetch all members anyway.
	var c ConversationDetails
	var name sql.NullString
	var ownerId sql.NullInt64
	err = db.c.QueryRow("SELECT id, name, is_group, photo, owner_id FROM conversations WHERE id = ?", conversationId).Scan(&c.ID, &name, &c.IsGroup, &c.Photo, &ownerId)
	if err != nil {
		return ConversationDetails{}, err
	}
	c.Name = name.String
	if ownerId.Valid {
		c.OwnerID = uint64(ownerId.Int64)
	}

	// Fetch Members
	mRows, err := db.c.Query("SELECT u.id, u.username, u.photo FROM users u JOIN conversation_members cm ON u.id = cm.user_id WHERE cm.conversation_id = ?", conversationId)
	if err != nil {
		return ConversationDetails{}, err
	}
	defer mRows.Close()
	for mRows.Next() {
		var u User
		if err := mRows.Scan(&u.ID, &u.Username, &u.Photo); err != nil {
			return ConversationDetails{}, err
		}
		c.Members = append(c.Members, u)
	}

	// Fix Name/Photo for 1-to-1 if missing
	if !c.IsGroup {
		for _, m := range c.Members {
			if m.ID != userId {
				c.Name = m.Username
				c.Photo = m.Photo
				break
			}
		}
		// If for some reason we are the only one (self-chat loop?), fallback to own name or handle gracefully.
		// If c.Name is still empty (e.g. self chat?), set to "Me" or own username.
		if c.Name == "" && len(c.Members) > 0 {
			c.Name = c.Members[0].Username
			c.Photo = c.Members[0].Photo
		}
	}

	// Fetch Messages
	msgRows, err := db.c.Query("SELECT id, conversation_id, sender_id, content, type, timestamp, received, read FROM messages WHERE conversation_id = ? ORDER BY timestamp ASC", conversationId)
	if err != nil {
		return ConversationDetails{}, err
	}
	defer msgRows.Close()
	for msgRows.Next() {
		var m Message
		var ts time.Time
		if err := msgRows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &ts, &m.Received, &m.Read); err != nil {
			return ConversationDetails{}, err
		}
		m.Timestamp = ts

		// Fetch reactions
		rRows, err := db.c.Query("SELECT id, message_id, user_id, emoji FROM reactions WHERE message_id = ?", m.ID)
		if err == nil {
			for rRows.Next() {
				var r Reaction
				rRows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji)
				m.Reactions = append(m.Reactions, r)
			}
			rRows.Close()
		}

		c.Messages = append(c.Messages, m)
	}

	return c, nil
}

// SetGroupName changes the group name
func (db *appdbimpl) SetGroupName(conversationId uint64, name string) error {
	res, err := db.c.Exec("UPDATE conversations SET name = ? WHERE id = ? AND is_group = 1", name, conversationId)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("conversation not found or not a group")
	}
	return nil
}

// SetGroupPhoto changes the group photo
func (db *appdbimpl) SetGroupPhoto(conversationId uint64, photo []byte) error {
	res, err := db.c.Exec("UPDATE conversations SET photo = ? WHERE id = ? AND is_group = 1", photo, conversationId)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("conversation not found or not a group")
	}
	return nil
}

// AddToGroup adds a user to a group
func (db *appdbimpl) AddToGroup(conversationId uint64, userId uint64) error {
	var isGroup bool
	err := db.c.QueryRow("SELECT is_group FROM conversations WHERE id = ?", conversationId).Scan(&isGroup)
	if err != nil {
		return err
	}
	if !isGroup {
		return errors.New("not a group")
	}

	_, err = db.c.Exec("INSERT INTO conversation_members (conversation_id, user_id, last_seen) VALUES (?, ?, ?)", conversationId, userId, time.Now())
	if err != nil {
		return fmt.Errorf("error adding member: %w", err)
	}
	return nil
}

// LeaveGroup removes a user from a group
func (db *appdbimpl) LeaveGroup(conversationId uint64, userId uint64) error {
	var isGroup bool
	err := db.c.QueryRow("SELECT is_group FROM conversations WHERE id = ?", conversationId).Scan(&isGroup)
	if err != nil {
		return err
	}
	if !isGroup {
		return errors.New("cannot leave a 1-on-1 conversation, just delete it or ignore it")
	}

	res, err := db.c.Exec("DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?", conversationId, userId)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not in group")
	}
	return nil
}
