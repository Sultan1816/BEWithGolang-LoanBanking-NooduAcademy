package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var ErrNotFound = errors.New("user tidak ditemukan")

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	IsActive     bool
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (store *Store) ByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := store.db.QueryRowContext(ctx,
		`SELECT id, username, email, COALESCE(password_hash, ''),
		        COALESCE(is_active = B'1', FALSE)
		 FROM users WHERE LOWER(email) = $1`,
		strings.ToLower(strings.TrimSpace(email)),
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (store *Store) ByID(ctx context.Context, id int64) (User, error) {
	var user User
	err := store.db.QueryRowContext(ctx,
		`SELECT id, username, email, COALESCE(is_active = B'1', FALSE)
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (store *Store) Create(ctx context.Context, username, email, passwordHash string) (User, error) {
	var user User
	err := store.db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password_hash, is_active) VALUES ($1, $2, $3, B'1')
		 RETURNING id, username, email, COALESCE(is_active = B'1', FALSE)`,
		strings.TrimSpace(username), strings.ToLower(strings.TrimSpace(email)), passwordHash,
	).Scan(&user.ID, &user.Username, &user.Email, &user.IsActive)
	return user, err
}

func (store *Store) UpdateProfile(ctx context.Context, id int64, username, email *string) (User, error) {
	usernameValue, emailValue := "", ""
	if username != nil {
		usernameValue = strings.TrimSpace(*username)
	}
	if email != nil {
		emailValue = strings.ToLower(strings.TrimSpace(*email))
	}

	var user User
	err := store.db.QueryRowContext(ctx,
		`UPDATE users
		 SET username = CASE WHEN $2 THEN $3 ELSE username END,
		     email = CASE WHEN $4 THEN $5 ELSE email END,
		     updateda_at = NOW()
		 WHERE id = $1 AND is_active = B'1'
		 RETURNING id, username, email, COALESCE(is_active = B'1', FALSE)`,
		id, username != nil, usernameValue, email != nil, emailValue,
	).Scan(&user.ID, &user.Username, &user.Email, &user.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (store *Store) Delete(ctx context.Context, id int64) error {
	result, err := store.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
