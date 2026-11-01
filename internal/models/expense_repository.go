package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

var ErrNotFound = errors.New("запись не найдена")

type ExpenseRepository struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Create(amount float64, description string, date time.Time, categoryID int) (int, error) {
	result, err := r.db.Exec(
		"INSERT INTO expenses (amount, description, date, category_id) VALUES (?, ?, ?, ?)",
		amount, description, date.Format(dateLayout), categoryID,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *ExpenseRepository) GetAll(filter ExpenseFilter) ([]Expense, error) {
	query := `SELECT e.id, e.amount, e.description, e.date, e.category_id, c.name
		FROM expenses e
		JOIN categories c ON c.id = e.category_id`

	var conditions []string
	var args []any

	if filter.CategoryID > 0 {
		conditions = append(conditions, "e.category_id = ?")
		args = append(args, filter.CategoryID)
	}
	if filter.DateFrom != "" {
		conditions = append(conditions, "e.date >= ?")
		args = append(args, filter.DateFrom)
	}
	if filter.DateTo != "" {
		conditions = append(conditions, "e.date <= ?")
		args = append(args, filter.DateTo)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY e.id"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		var dateStr string
		err := rows.Scan(&e.ID, &e.Amount, &e.Description, &dateStr, &e.CategoryID, &e.CategoryName)
		if err != nil {
			return nil, err
		}
		e.Date, err = time.Parse(dateLayout, dateStr)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return expenses, nil
}

func (r *ExpenseRepository) GetByID(id int) (Expense, error) {
	var e Expense
	var dateStr string

	err := r.db.QueryRow(
		`SELECT e.id, e.amount, e.description, e.date, e.category_id, c.name
		FROM expenses e
		JOIN categories c ON c.id = e.category_id
		WHERE e.id = ?`, id,
	).Scan(&e.ID, &e.Amount, &e.Description, &dateStr, &e.CategoryID, &e.CategoryName)
	if errors.Is(err, sql.ErrNoRows) {
		return Expense{}, ErrNotFound
	}
	if err != nil {
		return Expense{}, err
	}

	e.Date, err = time.Parse(dateLayout, dateStr)
	if err != nil {
		return Expense{}, err
	}
	return e, nil
}

func (r *ExpenseRepository) Update(id int, amount float64, description string, date time.Time, categoryID int) error {
	result, err := r.db.Exec(
		"UPDATE expenses SET amount = ?, description = ?, date = ?, category_id = ? WHERE id = ?",
		amount, description, date.Format(dateLayout), categoryID, id,
	)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ExpenseRepository) Delete(id int) error {
	result, err := r.db.Exec("DELETE FROM expenses WHERE id = ?", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
