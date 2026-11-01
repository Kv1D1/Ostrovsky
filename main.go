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

	app, err := handlers.New(repo, categoryRepo, "web/templates")
	if err != nil {
		log.Fatalf("ошибка загрузки шаблонов: %v", err)
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/about", app.About)
	mux.HandleFunc("/ping", app.Ping)

	mux.HandleFunc("GET /expenses", app.ExpensesList)
	mux.HandleFunc("GET /expenses/new", app.ExpenseNewForm)
	mux.HandleFunc("POST /expenses", app.ExpenseCreate)
	mux.HandleFunc("GET /expenses/{id}/edit", app.ExpenseEditForm)
	mux.HandleFunc("POST /expenses/{id}/update", app.ExpenseUpdate)
	mux.HandleFunc("POST /expenses/{id}/delete", app.ExpenseDelete)

	log.Println("сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
