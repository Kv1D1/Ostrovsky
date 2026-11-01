package models

import "time"

type Expense struct {
	ID           int
	Amount       float64
	Description  string
	Date         time.Time
	CategoryID   int
	CategoryName string
}

type ExpenseFilter struct {
	CategoryID int
	DateFrom   string
	DateTo     string
}
