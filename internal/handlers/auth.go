package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"expenses/internal/models"
)

// Свой тип для ключа, чтобы не пересечься с чужими ключами в контексте.
type ctxKey string

const userKey ctxKey = "user_id"

const cookieName = "session"
const sessionTTL = 7 * 24 * time.Hour

// userID достаёт id пользователя из контекста запроса. 0 — не вошёл.
func userID(r *http.Request) int {
	id, _ := r.Context().Value(userKey).(int)
	return id
}

// LoadUser — middleware для ВСЕХ страниц. Если есть живая сессия,
// кладёт id пользователя в контекст. Ничего не блокирует.
func (a *App) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err == nil {
			uid, err := a.sessions.GetUserID(c.Value)
			if err == nil {
				ctx := context.WithValue(r.Context(), userKey, uid)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireLogin — middleware для закрытых страниц.
func (a *App) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID(r) == 0 {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) startSession(w http.ResponseWriter, uid int) error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	id := hex.EncodeToString(b)
	exp := time.Now().Add(sessionTTL)

	if err := a.sessions.Create(id, uid, exp); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,                 // JS не сможет прочитать cookie
		SameSite: http.SameSiteLaxMode, // защита от CSRF
		// Secure: true, // включить, когда будет HTTPS
	})
	return nil
}

func (a *App) RegisterForm(w http.ResponseWriter, r *http.Request) {
	if userID(r) != 0 {
		http.Redirect(w, r, "/expenses", http.StatusSeeOther)
		return
	}
	a.render(w, r, http.StatusOK, "register", PageData{Title: "Регистрация", Active: "register"})
}

func (a *App) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "некорректные данные формы", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	pass := r.FormValue("password")

	fail := func(msg string) {
		a.render(w, r, http.StatusBadRequest, "register", PageData{
			Title: "Регистрация", Active: "register",
			Form: FormValues{Email: email}, Error: msg,
		})
	}

	if !strings.Contains(email, "@") {
		fail("Введите корректный email.")
		return
	}
	if len(pass) < 6 {
		fail("Пароль должен быть не короче 6 символов.")
		return
	}
	// bcrypt умеет работать максимум с 72 байтами
	if len(pass) > 72 {
		fail("Пароль слишком длинный (максимум 72 байта).")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		a.serverError(w, err)
		return
	}

	uid, err := a.users.Create(email, string(hash))
	if errors.Is(err, models.ErrEmailTaken) {
		fail("Пользователь с таким email уже есть.")
		return
	}
	if err != nil {
		a.serverError(w, err)
		return
	}

	// после регистрации сразу пускаем пользователя внутрь
	if err := a.startSession(w, uid); err != nil {
		a.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (a *App) LoginForm(w http.ResponseWriter, r *http.Request) {
	if userID(r) != 0 {
		http.Redirect(w, r, "/expenses", http.StatusSeeOther)
		return
	}
	a.render(w, r, http.StatusOK, "login", PageData{Title: "Вход", Active: "login"})
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "некорректные данные формы", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	pass := r.FormValue("password")

	// Одно и то же сообщение и для неверного email, и для неверного пароля,
	// чтобы нельзя было узнать, какие email зарегистрированы.
	fail := func() {
		a.render(w, r, http.StatusUnauthorized, "login", PageData{
			Title: "Вход", Active: "login",
			Form: FormValues{Email: email}, Error: "Неверный email или пароль.",
		})
	}

	u, err := a.users.GetByEmail(email)
	if errors.Is(err, models.ErrNotFound) {
		fail()
		return
	}
	if err != nil {
		a.serverError(w, err)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pass)); err != nil {
		fail()
		return
	}

	// заодно убираем старые просроченные сессии
	a.sessions.DeleteExpired()

	if err := a.startSession(w, u.ID); err != nil {
		a.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieName)
	if err == nil {
		// удаляем сессию на сервере — теперь cookie уже ничего не даёт
		if err := a.sessions.Delete(c.Value); err != nil {
			a.serverError(w, err)
			return
		}
	}

	// и просим браузер выкинуть cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
