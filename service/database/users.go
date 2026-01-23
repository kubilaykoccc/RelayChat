package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// DoLogin checks if a user exists. If not, creates one. Returns the user ID (which acts as the token).
func (db *appdbimpl) DoLogin(username string) (uint64, error) {
	var id uint64
	err := db.c.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("error querying user: %w", err)
	}

	// User not found, create new one
	res, err := db.c.Exec("INSERT INTO users (username) VALUES (?)", username)
	if err != nil {
		return 0, fmt.Errorf("error inserting user: %w", err)
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting last insert id: %w", err)
	}

	return uint64(lastId), nil
}

// SetMyUserName updates the user's username
func (db *appdbimpl) SetMyUserName(id uint64, name string) error {
	res, err := db.c.Exec("UPDATE users SET username = ? WHERE id = ?", name, id)
	if err != nil {
		return fmt.Errorf("error updating username: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}
	if rows == 0 {
		return errors.New("user not found or name unchanged")
	}
	return nil
}

// SetMyPhoto updates the user's photo
func (db *appdbimpl) SetMyPhoto(id uint64, photo []byte) error {
	res, err := db.c.Exec("UPDATE users SET photo = ? WHERE id = ?", photo, id)
	if err != nil {
		return fmt.Errorf("error updating photo: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

// SearchUsers finds users matching the query string
func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	rows, err := db.c.Query("SELECT id, username, photo FROM users WHERE username LIKE ?", "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("error searching users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Photo); err != nil {
			return nil, fmt.Errorf("error scanning user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}
	return users, nil
}

// GetUser retrieves a user by ID
func (db *appdbimpl) GetUser(id uint64) (User, error) {
	var u User
	err := db.c.QueryRow("SELECT id, username, photo FROM users WHERE id = ?", id).Scan(&u.ID, &u.Username, &u.Photo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, errors.New("user not found")
		}
		return User{}, fmt.Errorf("error getting user: %w", err)
	}
	return u, nil
}
