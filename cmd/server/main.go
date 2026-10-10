package main

import (
	"log"
	"net/http"

	"expenses/internal/handlers"
	"expenses/internal/models"
)

func main() {

	store := models.NewExpenseStore()

	app, err := handlers.New(store, "web/templates")
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

	log.Println("сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
