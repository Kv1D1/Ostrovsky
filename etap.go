package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	x := http.NewServeMux()

	y := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Добро пожаловать на главную страницу.")
	}

	z := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Это учебный проект: простой HTTP-сервер на Go (net/http).")
	}

	p := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "метод не разрешён", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "pong")
	}

	x.HandleFunc("/", y)
	x.HandleFunc("/about", z)
	x.HandleFunc("/ping", p)

	log.Println("сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", x))
}
