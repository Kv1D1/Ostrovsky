package models

import (
	"database/sql"
	"errors"
	"strings"
)

var ErrEmailTaken = errors.New("email уже занят")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(email, hash string) (int, error) {
	res, err := r.db.Exec("INSERT INTO users (email, password_hash) VALUES (?, ?)", email, hash)
	if err != nil {
		// email у нас UNIQUE, значит это дубликат
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, ErrEmailTaken
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *UserRepository) GetByEmail(email string) (User, error) {
	var u User
	err := r.db.QueryRow(
		"SELECT id, email, password_hash FROM users WHERE email = ?", email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}
