package models

import (
	"database/sql"
	"errors"
	"time"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(id string, userID int, expires time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)",
		id, userID, expires.Unix(),
	)
	return err
}

// Срок действия проверяется прямо в запросе: просроченная сессия просто не найдётся.
func (r *SessionRepository) GetUserID(id string) (int, error) {
	var uid int
	err := r.db.QueryRow(
		"SELECT user_id FROM sessions WHERE id = ? AND expires_at > ?",
		id, time.Now().Unix(),
	).Scan(&uid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return uid, nil
}

func (r *SessionRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// Чистим мусор: просроченные сессии сами из таблицы не пропадут.
func (r *SessionRepository) DeleteExpired() error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now().Unix())
	return err
}
