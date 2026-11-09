package handlers

import (
	"bytes"
	"errors"
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
	Title      string
	Active     string
	LoggedIn   bool
	Expenses   []models.Expense
	Total      float64
	Categories []models.Category
	Filter     models.ExpenseFilter

	ID    int
	Form  FormValues
	Error string
}

type FormValues struct {
	Email       string
	Amount      string
	Description string
	Date        string
	CategoryID  int
}

type App struct {
	repo       *models.ExpenseRepository
	categories *models.CategoryRepository
	users      *models.UserRepository
	sessions   *models.SessionRepository
	pages      map[string]*template.Template
}

func New(
	repo *models.ExpenseRepository,
	categories *models.CategoryRepository,
	users *models.UserRepository,
	sessions *models.SessionRepository,
	templatesDir string,
) (*App, error) {
	layout := filepath.Join(templatesDir, "layout.html")
	names := []string{"home", "about", "expenses", "expense_new", "expense_edit", "login", "register"}

	pages := make(map[string]*template.Template, len(names))
	for _, name := range names {
		page := filepath.Join(templatesDir, name+".html")
		t, err := template.ParseFiles(layout, page)
		if err != nil {
			return nil, fmt.Errorf("шаблон %s: %w", name, err)
		}
		pages[name] = t
	}
	return &App{
		repo:       repo,
		categories: categories,
		users:      users,
		sessions:   sessions,
		pages:      pages,
	}, nil
}

func (a *App) render(w http.ResponseWriter, r *http.Request, status int, page string, data PageData) {
	t, ok := a.pages[page]
	if !ok {
		http.Error(w, "шаблон не найден", http.StatusInternalServerError)
		return
	}
	// layout по этому флагу решает, что показать в меню
	data.LoggedIn = userID(r) != 0

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

func (a *App) renderWithCategories(w http.ResponseWriter, r *http.Request, status int, page string, data PageData) {
	categories, err := a.categories.GetAll()
	if err != nil {
		a.serverError(w, err)
		return
	}
	data.Categories = categories
	a.render(w, r, status, page, data)
}

func (a *App) serverError(w http.ResponseWriter, err error) {
	log.Printf("ошибка: %v", err)
	http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
}

func readForm(r *http.Request) FormValues {
	categoryID, _ := strconv.Atoi(r.FormValue("category"))
	return FormValues{
		Amount:      strings.TrimSpace(r.FormValue("amount")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Date:        strings.TrimSpace(r.FormValue("date")),
		CategoryID:  categoryID,
	}
}

func (a *App) checkForm(form FormValues) (float64, time.Time, string) {
	amount, err := strconv.ParseFloat(strings.ReplaceAll(form.Amount, ",", "."), 64)
	if err != nil || amount <= 0 {
		return 0, time.Time{}, "Сумма должна быть положительным числом."
	}
	if form.Description == "" {
		return 0, time.Time{}, "Описание не может быть пустым."
	}
	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		return 0, time.Time{}, "Укажите корректную дату."
	}
	if _, err := a.categories.GetByID(form.CategoryID); err != nil {
		return 0, time.Time{}, "Выберите категорию из списка."
	}
	return amount, date, ""
}

func readDate(value string) string {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return ""
	}
	return value
}

func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, http.StatusOK, "home", PageData{Title: "Главная", Active: "home"})
}

func (a *App) About(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, http.StatusOK, "about", PageData{Title: "О проекте", Active: "about"})
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
	uid := userID(r)
	query := r.URL.Query()

	categoryID, _ := strconv.Atoi(query.Get("category"))
	filter := models.ExpenseFilter{
		CategoryID: categoryID,
		DateFrom:   readDate(query.Get("from")),
		DateTo:     readDate(query.Get("to")),
	}

	expenses, err := a.repo.GetAll(uid, filter)
	if err != nil {
		a.serverError(w, err)
		return
	}

	var total float64
	for _, e := range expenses {
		total += e.Amount
	}

	a.renderWithCategories(w, r, http.StatusOK, "expenses", PageData{
		Title:    "Список трат",
		Active:   "expenses",
		Expenses: expenses,
		Total:    total,
		Filter:   filter,
	})
}

func (a *App) ExpenseNewForm(w http.ResponseWriter, r *http.Request) {
	a.renderWithCategories(w, r, http.StatusOK, "expense_new", PageData{
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

	form := readForm(r)

	amount, date, msg := a.checkForm(form)
	if msg != "" {
		a.renderWithCategories(w, r, http.StatusBadRequest, "expense_new", PageData{
			Title: "Новая трата", Active: "new", Form: form, Error: msg,
		})
		return
	}

	if _, err := a.repo.Create(userID(r), amount, form.Description, date, form.CategoryID); err != nil {
		a.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (a *App) ExpenseEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// чужая трата здесь вернёт ErrNotFound, то есть 404
	expense, err := a.repo.GetByID(userID(r), id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.serverError(w, err)
		return
	}

	a.renderWithCategories(w, r, http.StatusOK, "expense_edit", PageData{
		Title:  "Редактирование траты",
		Active: "expenses",
		ID:     expense.ID,
		Form: FormValues{
			Amount:      strconv.FormatFloat(expense.Amount, 'f', 2, 64),
			Description: expense.Description,
			Date:        expense.Date.Format("2006-01-02"),
			CategoryID:  expense.CategoryID,
		},
	})
}

func (a *App) ExpenseUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "некорректные данные формы", http.StatusBadRequest)
		return
	}

	form := readForm(r)

	amount, date, msg := a.checkForm(form)
	if msg != "" {
		a.renderWithCategories(w, r, http.StatusBadRequest, "expense_edit", PageData{
			Title: "Редактирование траты", Active: "expenses", ID: id, Form: form, Error: msg,
		})
		return
	}

	err = a.repo.Update(userID(r), id, amount, form.Description, date, form.CategoryID)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (a *App) ExpenseDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = a.repo.Delete(userID(r), id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}
