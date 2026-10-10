package models

import (
	"sync"
	"time"
)

type Expense struct {
	ID          int
	Amount      float64   // сумма
	Description string    // описание
	Date        time.Time // дата траты
}

type ExpenseStore struct {
	mu       sync.RWMutex
	expenses []Expense
	nextID   int
}

func NewExpenseStore() *ExpenseStore {
	return &ExpenseStore{nextID: 1}
}

func (s *ExpenseStore) Add(amount float64, description string, date time.Time) Expense {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := Expense{ID: s.nextID, Amount: amount, Description: description, Date: date}
	s.nextID++
	s.expenses = append(s.expenses, e)
	return e
}

func (s *ExpenseStore) All() []Expense {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Expense, len(s.expenses))
	copy(out, s.expenses)
	return out
}

func (s *ExpenseStore) Total() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sum float64
	for _, e := range s.expenses {
		sum += e.Amount
	}
	return sum
}
