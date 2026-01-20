package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Subscription struct {
	ID int64 `json:"id"`
	SubscriptionSpec
}

type SubscriptionSpec struct {
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   Date      `json:"start_date"`
	EndDate     Date      `json:"end_date"`
}

type Date struct {
	Month int `json:"month"`
	Year  int `json:"year"`
}

// Add another date to current
func (d *Date) AddDate(years, months int) Date {
	var newD Date

	newD.Year = d.Year + years
	newD.Year += months / 12

	newD.Month = d.Month + months%12

	if newD.Month > 12 {
		newD.Year += 1
		newD.Month -= 12
	}

	if newD.Month <= 0 {
		newD.Year -= 1
		newD.Month += 12
	}

	return newD
}

// Check if other date greater than current
func (d *Date) GreaterThan(other Date) bool {
	if d.Year > other.Year {
		return true
	}
	if d.Year == other.Year && d.Month > other.Month {
		return true
	}
	return false
}

// Convert to string representation
func (d *Date) ToString() string {
	if d.Month > 9 {
		return fmt.Sprintf("%d-%d", d.Month, d.Year)
	}
	return fmt.Sprintf("0%d-%d", d.Month, d.Year)
}

// Convert to string in ISO format YYYY-MM-DD
func (d *Date) ToStringISO() string {
	if d.Month > 9 {
		return fmt.Sprintf("%d-%d-01", d.Year, d.Month)
	}
	return fmt.Sprintf("%d-0%d-01", d.Year, d.Month)
}

// Check if equal to another date
func (d *Date) EqualTo(other Date) bool {
	return d.Month == other.Month && d.Year == other.Year
}

// Construct from string; string must be in "MM-YYYY" format!
func DateFromString(str string) (Date, error) {
	items := strings.Split(str, "-")
	if len(items) != 2 {
		return Date{}, errors.New("invalid date string format")
	}

	month, err := strconv.Atoi(items[0])
	if err != nil {
		return Date{}, errors.New("invalid month")
	}

	year, err := strconv.Atoi(items[1])
	if err != nil {
		return Date{}, errors.New("invalid year")
	}

	return Date{Month: month, Year: year}, nil
}

// Construct from string in ISO format YYYY-MM-DD
func DateFromStringISO(str string) (Date, error) {
	items := strings.Split(str, "-")
	if len(items) != 3 {
		return Date{}, errors.New("invalid date string ISO format")
	}

	year, err := strconv.Atoi(items[0])
	if err != nil {
		return Date{}, errors.New("invalid year")
	}

	month, err := strconv.Atoi(items[1])
	if err != nil {
		return Date{}, errors.New("invalid month")
	}

	return Date{Month: month, Year: year}, nil
}
