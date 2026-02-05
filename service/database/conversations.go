package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreateConversation creates a new conversation (could be direct or group, starting as direct usually)
func (db *appdbimpl) CreateConversation(ownerId uint64, otherUserId uint64) (Conversation, error) {
	// Check if other user exists
	var exists int
	checkErr := db.c.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", otherUserId).Scan(&exists)
	if checkErr != nil {
		return Conversation{}, fmt.Errorf("error checking user existence: %w", checkErr)
	}
	if exists == 0 {
		return Conversation{}, errors.New("target user not found")
	}

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
	defer func() {
		_ = tx.Rollback()
	}()

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
	defer func() {
		_ = tx.Rollback()
	}()

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
	res, err := db.c.Exec("UPDATE conversation_members SET last_delivered = ? WHERE user_id = ? AND last_delivered < ?", time.Now(), userId, time.Now().Add(-10*time.Second))
	if err == nil {
		rows, _ := res.RowsAffected()
		if rows > 0 {
			// Only run the heavy status updates if we actually touched the timestamp
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
			_, _ = db.c.Exec(updateReceived, userId)
		}
	}

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
	rows, err := db.c.Query(query, userId, userId)
	if err != nil {
		return nil, fmt.Errorf("error getting conversations: %w", err)
	}
	defer rows.Close()

	var results []ConversationUnread
	// Map to quickly assign auxiliary data
	convMap := make(map[uint64]*ConversationUnread)
	var convIDs []interface{} // built for IN clause

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

		results = append(results, c)

		convIDs = append(convIDs, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}
	// Pointers map for easy lookup
	for i := range results {
		convMap[results[i].ID] = &results[i]
	}

	if len(results) > 0 {

		placeholders := ""
		for i := 0; i < len(convIDs); i++ {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}

		ucQuery := fmt.Sprintf(`
			SELECT conversation_id, COUNT(*) 
			FROM messages 
			WHERE sender_id != ? AND read = 0 AND conversation_id IN (%s)
			GROUP BY conversation_id
		`, placeholders)

		ucArgs := append([]interface{}{userId}, convIDs...)
		ucRows, err := db.c.Query(ucQuery, ucArgs...)
		if err == nil {
			defer ucRows.Close()
			for ucRows.Next() {
				var cid uint64
				var count int
				if err := ucRows.Scan(&cid, &count); err == nil {
					if c, ok := convMap[cid]; ok {
						c.UnreadCount = count
					}
				}
			}
			if err := ucRows.Err(); err != nil {
				return nil, fmt.Errorf("error iterating unread counts: %w", err)
			}
		}

		lmQuery := fmt.Sprintf(`
			SELECT m.id, m.conversation_id, m.sender_id, m.content, m.type, m.timestamp, m.received, m.read
			FROM messages m
			WHERE m.id IN (
				SELECT MAX(id) FROM messages WHERE conversation_id IN (%s) GROUP BY conversation_id
			)
		`, placeholders)

		lmRows, err := db.c.Query(lmQuery, convIDs...)
		if err == nil {
			defer lmRows.Close()
			for lmRows.Next() {
				var m Message
				var ts time.Time
				if err := lmRows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &ts, &m.Received, &m.Read); err == nil {
					m.Timestamp = ts
					if c, ok := convMap[m.ConversationID]; ok {
						c.LastMessage = &m
					}
				}
			}
			if err := lmRows.Err(); err != nil {
				return nil, fmt.Errorf("error iterating last messages: %w", err)
			}
		}
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

	res, err := db.c.Exec("UPDATE conversation_members SET last_seen = ? WHERE conversation_id = ? AND user_id = ? AND last_seen < ?", time.Now(), conversationId, userId, time.Now().Add(-10*time.Second))
	if err == nil {
		rows, _ := res.RowsAffected()
		if rows > 0 {
			// Also update last_delivered if we updated last_seen
			_, _ = db.c.Exec("UPDATE conversation_members SET last_delivered = ? WHERE conversation_id = ? AND user_id = ?", time.Now(), conversationId, userId)
		}
	}

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
	if err := mRows.Err(); err != nil {
		return ConversationDetails{}, fmt.Errorf("error iterating members: %w", err)
	}

	if !c.IsGroup {
		for _, m := range c.Members {
			if m.ID != userId {
				c.Name = m.Username
				c.Photo = m.Photo
				break
			}
		}

		if c.Name == "" && len(c.Members) > 0 {
			c.Name = c.Members[0].Username
			c.Photo = c.Members[0].Photo
		}
	}

	// Fetch Messages first (without reactions inside loop)
	query := `
		SELECT 
			m.id, m.conversation_id, m.sender_id, m.content, m.type, m.timestamp, m.received, m.read, m.photo, m.reply_to_id,
			r.id, r.sender_id, r.content, r.type
		FROM messages m
		LEFT JOIN messages r ON m.reply_to_id = r.id
		WHERE m.conversation_id = ? 
		ORDER BY m.timestamp ASC
	`
	msgRows, err := db.c.Query(query, conversationId)
	if err != nil {
		return ConversationDetails{}, err
	}
	defer msgRows.Close()

	var messagesList []Message
	// Map to keep track of message indices to attach reactions later
	msgMap := make(map[uint64]*Message)

	for msgRows.Next() {
		var m Message
		var ts time.Time
		var replyID sql.NullInt64
		var rID sql.NullInt64
		var rSenderID sql.NullInt64
		var rContent sql.NullString
		var rType sql.NullString

		if err := msgRows.Scan(
			&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &ts, &m.Received, &m.Read, &m.Photo, &replyID,
			&rID, &rSenderID, &rContent, &rType,
		); err != nil {
			return ConversationDetails{}, err
		}
		m.Timestamp = ts
		if replyID.Valid {
			rid := uint64(replyID.Int64)
			m.ReplyToID = &rid
		}

		// Populate ReplyTo if exists
		if rID.Valid {
			reply := Message{
				ID:       uint64(rID.Int64),
				SenderID: uint64(rSenderID.Int64),
				Content:  rContent.String,
				Type:     rType.String,
			}
			m.ReplyTo = &reply
		}

		m.Reactions = []Reaction{} // Initialize empty
		messagesList = append(messagesList, m)
	}
	if err := msgRows.Err(); err != nil {
		return ConversationDetails{}, fmt.Errorf("error iterating messages: %w", err)
	}

	// Create pointers map for second pass (reactions)
	for i := range messagesList {
		msgMap[messagesList[i].ID] = &messagesList[i]
	}

	// Efficient Batch Fetch Reactions
	// We fetch all reactions for messages in this conversation
	reactionQuery := `
		SELECT r.id, r.message_id, r.user_id, r.emoji
		FROM reactions r
		JOIN messages m ON r.message_id = m.id
		WHERE m.conversation_id = ?
	`
	rRows, err := db.c.Query(reactionQuery, conversationId)
	if err == nil {
		defer rRows.Close()
		for rRows.Next() {
			var r Reaction
			if err := rRows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji); err == nil {
				if msg, exists := msgMap[r.MessageID]; exists {
					msg.Reactions = append(msg.Reactions, r)
				}
			}
		}
		if err := rRows.Err(); err != nil {
			// Non-critical? But good to log or return
			return ConversationDetails{}, fmt.Errorf("error iterating reactions: %w", err)
		}
	}

	c.Messages = messagesList

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
