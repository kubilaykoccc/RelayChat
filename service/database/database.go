/*
Package database is the middleware between the app database and the code. All data (de)serialization (save/load) from a
persistent database are handled here. Database specific logic should never escape this package.

To use this package you need to apply migrations to the database if needed/wanted, connect to it (using the database
data source name from config), and then initialize an instance of AppDatabase from the DB connection.

For example, this code adds a parameter in `webapi` executable for the database data source name (add it to the
main.WebAPIConfiguration structure):

	DB struct {
		Filename string `conf:""`
	}

This is an example on how to migrate the DB and connect to it:

	// Start Database
	logger.Println("initializing database support")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		logger.WithError(err).Error("error opening SQLite DB")
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		logger.Debug("database stopping")
		_ = db.Close()
	}()

Then you can initialize the AppDatabase and pass it to the api package.
*/
package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	// User
	DoLogin(username string) (string, error)
	SetMyUserName(id string, name string) error
	SetMyPhoto(id string, photo []byte) error
	SearchUsers(query string) ([]User, error)
	GetUser(id string) (User, error) // Helper to get user details

	// Conversation
	GetMyConversations(userId string) ([]ConversationUnread, error)
	GetConversation(conversationId string, userId string) (ConversationDetails, error)
	SetGroupName(conversationId string, name string, userId string) error
	SetGroupPhoto(conversationId string, photo []byte, userId string) error
	AddToGroup(conversationId string, userId string) error
	LeaveGroup(conversationId string, userId string) error
	CreateConversation(ownerId string, otherUserId string) (Conversation, error)
	CreateGroup(name string, ownerId string) (Conversation, error)

	// Message
	SendMessage(message Message) (Message, error)
	ForwardMessage(conversationId string, forwardedMessageId string, senderId string) (Message, error)
	DeleteMessage(messageId string, userId string) error

	// Reaction
	CommentMessage(reaction Reaction) (Reaction, error)
	UncommentMessage(reactionId string, userId string) error

	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("error enabling foreign keys: %w", err)
	}

	// Set busy_timeout to 5000ms to avoid "database is locked" errors during concurrent updates
	if _, err := db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		return nil, fmt.Errorf("error setting busy_timeout: %w", err)
	}

	// Create tables
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			photo BLOB
		);`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			name TEXT,
			is_group BOOLEAN NOT NULL,
			photo BLOB,
			owner_id TEXT,
			FOREIGN KEY (owner_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS conversation_members (
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			last_seen DATETIME NOT NULL,
			last_delivered DATETIME DEFAULT '1970-01-01 00:00:00',
			PRIMARY KEY (conversation_id, user_id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			content TEXT,
			type TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			received BOOLEAN DEFAULT FALSE,
			read BOOLEAN DEFAULT FALSE,
			reply_to_id TEXT,
			photo BLOB,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (sender_id) REFERENCES users(id),
			FOREIGN KEY (reply_to_id) REFERENCES messages(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS reactions (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			emoji TEXT NOT NULL,
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
	}

	for _, stmt := range tables {
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("error creating database structure: %w", err)
		}
	}

	// Performance Indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_messages_conv_ts ON messages(conversation_id, timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_reactions_msg_id ON reactions(message_id);",
		"CREATE INDEX IF NOT EXISTS idx_members_user_id ON conversation_members(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_members_conv_user ON conversation_members(conversation_id, user_id);",
	}
	for _, stmt := range indexes {
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("error creating index: %w", err)
		}
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
