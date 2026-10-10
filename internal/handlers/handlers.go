package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"expenses/internal/models"
)

type PageData struct {
	Title    string
	Active   string // подсветка пункта меню
	Expenses []models.Expense
	Total    float64

	Form  FormValues
	Error string
}

type FormValues struct {
	Amount      string
	Description string
	Date        string
}

type App struct {
	store *models.ExpenseStore
	pages map[string]*template.Template
}

func New(store *models.ExpenseStore, templatesDir string) (*App, error) {
	layout := filepath.Join(templatesDir, "layout.html")
	names := []string{"home", "about", "expenses", "expense_new"}

	pages := make(map[string]*template.Template, len(names))
	for _, name := range names {
		page := filepath.Join(templatesDir, name+".html")
		t, err := template.ParseFiles(layout, page)
		if err != nil {
			return nil, fmt.Errorf("шаблон %s: %w", name, err)
		}
		pages[name] = t
	}
	return &App{store: store, pages: pages}, nil
}

func (a *App) render(w http.ResponseWriter, status int, page string, data PageData) {
	t, ok := a.pages[page]
	if !ok {
		http.Error(w, "шаблон не найден", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("ошибка рендера %s: %v", page, err)
		http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	a.render(w, http.StatusOK, "home", PageData{Title: "Главная", Active: "home"})
}

func (a *App) About(w http.ResponseWriter, r *http.Request) {
	a.render(w, http.StatusOK, "about", PageData{Title: "О проекте", Active: "about"})
}

func (a *App) Ping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "метод не разрешён", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "pong")
}

func (a *App) ExpensesList(w http.ResponseWriter, r *http.Request) {
	a.render(w, http.StatusOK, "expenses", PageData{
		Title:    "Список трат",
		Active:   "expenses",
		Expenses: a.store.All(),
		Total:    a.store.Total(),
	})
}

func (a *App) ExpenseNewForm(w http.ResponseWriter, r *http.Request) {
	a.render(w, http.StatusOK, "expense_new", PageData{
		Title:  "Новая трата",
		Active: "new",
		Form:   FormValues{Date: time.Now().Format("2006-01-02")},
	})
}

func (a *App) ExpenseCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "некорректные данные формы", http.StatusBadRequest)
		return
	}

	form := FormValues{
		Amount:      strings.TrimSpace(r.FormValue("amount")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Date:        strings.TrimSpace(r.FormValue("date")),
	}

	fail := func(msg string) {
		a.render(w, http.StatusBadRequest, "expense_new", PageData{
			Title: "Новая трата", Active: "new", Form: form, Error: msg,
		})
	}

	amount, err := strconv.ParseFloat(strings.ReplaceAll(form.Amount, ",", "."), 64)
	if err != nil || amount <= 0 {
		fail("Сумма должна быть положительным числом.")
		return
	}
	if form.Description == "" {
		fail("Описание не может быть пустым.")
		return
	}
	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		fail("Укажите корректную дату.")
		return
	}

	a.store.Add(amount, form.Description, date)

	http.Redirect(w, r, "/expenses", http.StatusSeeOther) // 303
}
