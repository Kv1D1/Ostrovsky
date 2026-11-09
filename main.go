package main

import (
	"log"
	"net/http"

	"expenses/internal/handlers"
	"expenses/internal/models"
)

func main() {
	db, err := models.OpenDB("expenses.db", "migrations")
	if err != nil {
		log.Fatalf("ошибка открытия базы данных: %v", err)
	}
	defer db.Close()

	repo := models.NewExpenseRepository(db)
	categoryRepo := models.NewCategoryRepository(db)
	userRepo := models.NewUserRepository(db)
	sessionRepo := models.NewSessionRepository(db)

	app, err := handlers.New(repo, categoryRepo, userRepo, sessionRepo, "web/templates")
	if err != nil {
		log.Fatalf("ошибка загрузки шаблонов: %v", err)
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// закрытая страница = обычный обработчик, завёрнутый в RequireLogin
	closed := func(h http.HandlerFunc) http.Handler {
		return app.RequireLogin(h)
	}

	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/about", app.About)
	mux.HandleFunc("/ping", app.Ping)

	mux.HandleFunc("GET /register", app.RegisterForm)
	mux.HandleFunc("POST /register", app.Register)
	mux.HandleFunc("GET /login", app.LoginForm)
	mux.HandleFunc("POST /login", app.Login)
	mux.HandleFunc("POST /logout", app.Logout)

	mux.Handle("GET /expenses", closed(app.ExpensesList))
	mux.Handle("GET /expenses/new", closed(app.ExpenseNewForm))
	mux.Handle("POST /expenses", closed(app.ExpenseCreate))
	mux.Handle("GET /expenses/{id}/edit", closed(app.ExpenseEditForm))
	mux.Handle("POST /expenses/{id}/update", closed(app.ExpenseUpdate))
	mux.Handle("POST /expenses/{id}/delete", closed(app.ExpenseDelete))

	log.Println("сервер запущен на http://localhost:8080")
	// LoadUser оборачивает весь mux: сначала узнаём пользователя, потом маршрутизируем
	log.Fatal(http.ListenAndServe(":8080", app.LoadUser(mux)))
}
