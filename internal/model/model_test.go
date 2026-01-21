package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddDate(t *testing.T) {
	tests := []struct {
		name     string
		date     Date
		years    int
		months   int
		expected Date
	}{
		// Base cases
		{
			name:     "Add months",
			date:     Date{Month: 3, Year: 2023},
			years:    0,
			months:   2,
			expected: Date{Month: 5, Year: 2023},
		},
		{
			name:     "Add years",
			date:     Date{Month: 6, Year: 2020},
			years:    3,
			months:   0,
			expected: Date{Month: 6, Year: 2023},
		},
		{
			name:     "Add years and months",
			date:     Date{Month: 8, Year: 2021},
			years:    2,
			months:   4,
			expected: Date{Month: 12, Year: 2023},
		},

		// Over 12 months
		{
			name:     "Transition through a year when adding a month",
			date:     Date{Month: 10, Year: 2023},
			years:    0,
			months:   5,
			expected: Date{Month: 3, Year: 2024},
		},
		{
			name:     "Add 12 months",
			date:     Date{Month: 5, Year: 2022},
			years:    0,
			months:   12,
			expected: Date{Month: 5, Year: 2023},
		},
		{
			name:     "Add more then 12 months",
			date:     Date{Month: 7, Year: 2021},
			years:    0,
			months:   18,
			expected: Date{Month: 1, Year: 2023},
		},

		// Few years and months (> 12)
		{
			name:     "Difficult transition",
			date:     Date{Month: 11, Year: 2020},
			years:    2,
			months:   6,
			expected: Date{Month: 5, Year: 2023},
		},

		// Negative values
		{
			name:     "Subtract months",
			date:     Date{Month: 5, Year: 2023},
			years:    0,
			months:   -2,
			expected: Date{Month: 3, Year: 2023},
		},
		{
			name:     "Subtract years",
			date:     Date{Month: 6, Year: 2023},
			years:    -3,
			months:   0,
			expected: Date{Month: 6, Year: 2020},
		},
		{
			name:     "Subtract years and months",
			date:     Date{Month: 3, Year: 2023},
			years:    -2,
			months:   -4,
			expected: Date{Month: 11, Year: 2020},
		},

		// Go over 0 months (negative values)
		{
			name:     "Transition through a year when subtracting months",
			date:     Date{Month: 2, Year: 2023},
			years:    0,
			months:   -5,
			expected: Date{Month: 9, Year: 2022},
		},
		{
			name:     "Subtract 12 months",
			date:     Date{Month: 5, Year: 2023},
			years:    0,
			months:   -12,
			expected: Date{Month: 5, Year: 2022},
		},
		{
			name:     "Subtract > 12 months",
			date:     Date{Month: 3, Year: 2023},
			years:    0,
			months:   -18,
			expected: Date{Month: 9, Year: 2021},
		},

		// Few negative years and months
		{
			name:     "Complex transition with negative values",
			date:     Date{Month: 5, Year: 2023},
			years:    -2,
			months:   -6,
			expected: Date{Month: 11, Year: 2020},
		},

		// Bound cases
		{
			name:     "January with subtraction",
			date:     Date{Month: 1, Year: 2023},
			years:    0,
			months:   -1,
			expected: Date{Month: 12, Year: 2022},
		},
		{
			name:     "December with addition",
			date:     Date{Month: 12, Year: 2022},
			years:    0,
			months:   1,
			expected: Date{Month: 1, Year: 2023},
		},
		{
			name:     "Add 0 years and 0 months",
			date:     Date{Month: 7, Year: 2023},
			years:    0,
			months:   0,
			expected: Date{Month: 7, Year: 2023},
		},

		// Big values
		{
			name:     "Add many years",
			date:     Date{Month: 8, Year: 2000},
			years:    25,
			months:   0,
			expected: Date{Month: 8, Year: 2025},
		},
		{
			name:     "Add many months",
			date:     Date{Month: 1, Year: 2020},
			years:    0,
			months:   48,
			expected: Date{Month: 1, Year: 2024},
		},

		// Positive and negative combinations
		{
			name:     "Add years, subtract months",
			date:     Date{Month: 7, Year: 2020},
			years:    3,
			months:   -2,
			expected: Date{Month: 5, Year: 2023},
		},
		{
			name:     "Subtract years, add months",
			date:     Date{Month: 3, Year: 2023},
			years:    -1,
			months:   10,
			expected: Date{Month: 1, Year: 2023},
		},

		// Specific cases with months
		{
			name:     "Substract > 12 months from 1th month",
			date:     Date{Month: 1, Year: 2023},
			years:    0,
			months:   -15,
			expected: Date{Month: 10, Year: 2021},
		},
		{
			name:     "Add many months",
			date:     Date{Month: 6, Year: 2000},
			years:    0,
			months:   500, // 41 год и 8 months
			expected: Date{Month: 2, Year: 2042},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.date.AddDate(tt.years, tt.months)

			if result != tt.expected {
				t.Errorf("AddDate(%v, %d years, %d months) = %v, expected %v",
					tt.date, tt.years, tt.months, result, tt.expected)
			}
		})
	}
}

func TestAddDateImmutability(t *testing.T) {
	original := Date{Month: 5, Year: 2023}
	result := original.AddDate(1, 1)

	if original.Month != 5 || original.Year != 2023 {
		t.Errorf("Original date was modified: got %v, expected Month: 5, Year: 2023", original)
	}

	expected := Date{Month: 6, Year: 2024}
	if result != expected {
		t.Errorf("AddDate(1, 1) = %v, expected %v", result, expected)
	}
}

func TestGreaterThan(t *testing.T) {
	cases := []struct {
		name     string
		date1    Date
		date2    Date
		expected bool
	}{
		{
			name:     "Different attrs",
			date1:    Date{Month: 5, Year: 2023},
			date2:    Date{Month: 3, Year: 2024},
			expected: true,
		},
		{
			name:     "Same years",
			date1:    Date{Month: 5, Year: 2023},
			date2:    Date{Month: 7, Year: 2023},
			expected: true,
		},
		{
			name:     "Not greater 1",
			date1:    Date{Month: 5, Year: 2023},
			date2:    Date{Month: 7, Year: 2022},
			expected: false,
		},
		{
			name:     "Not greater 2",
			date1:    Date{Month: 5, Year: 2023},
			date2:    Date{Month: 1, Year: 2022},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			greater := tc.date2.GreaterThan(tc.date1)
			assert.Equal(t, tc.expected, greater)
		})
	}
}

func TestDateFromString(t *testing.T) {
	cases := []struct {
		name    string
		dateStr string
		date    Date
		errMsg  string
	}{
		{
			name:    "Success",
			dateStr: "05-2023",
			date:    Date{Month: 5, Year: 2023},
			errMsg:  "",
		},
		{
			name:    "Too much date elements",
			dateStr: "01-05-2023",
			date:    Date{},
			errMsg:  "invalid date string format",
		},
		{
			name:    "Invalid month",
			dateStr: "trash-2023",
			date:    Date{},
			errMsg:  "invalid month",
		},
		{
			name:    "Invalid year",
			dateStr: "05-trash",
			date:    Date{},
			errMsg:  "invalid year",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actualDate, err := DateFromString(tc.dateStr)
			assert.Equal(t, tc.date, actualDate)

			if tc.errMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}

func TestDateFromStringISO(t *testing.T) {
	cases := []struct {
		name    string
		dateStr string
		date    Date
		errMsg  string
	}{
		{
			name:    "Success",
			dateStr: "2023-05-01",
			date:    Date{Month: 5, Year: 2023},
			errMsg:  "",
		},
		{
			name:    "Not enough date elements",
			dateStr: "05-2023",
			date:    Date{},
			errMsg:  "invalid date string ISO format",
		},
		{
			name:    "Invalid month",
			dateStr: "2023-trash-01",
			date:    Date{},
			errMsg:  "invalid month",
		},
		{
			name:    "Invalid year",
			dateStr: "trash-05-01",
			date:    Date{},
			errMsg:  "invalid year",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actualDate, err := DateFromStringISO(tc.dateStr)
			assert.Equal(t, tc.date, actualDate)

			if tc.errMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}

func TestToString(t *testing.T) {
	t.Run("Month less than 9", func(t *testing.T) {
		date := Date{Month: 5, Year: 2025}
		dateStr := date.ToString()

		assert.Equal(t, "05-2025", dateStr)
	})

	t.Run("Month more than 9", func(t *testing.T) {
		date := Date{Month: 11, Year: 2025}
		dateStr := date.ToString()

		assert.Equal(t, "11-2025", dateStr)
	})
}

func TestToStringISO(t *testing.T) {
	t.Run("Month less than 9", func(t *testing.T) {
		date := Date{Month: 5, Year: 2025}
		dateStr := date.ToStringISO()

		assert.Equal(t, "2025-05-01", dateStr)
	})

	t.Run("Month more than 9", func(t *testing.T) {
		date := Date{Month: 11, Year: 2025}
		dateStr := date.ToStringISO()

		assert.Equal(t, "2025-11-01", dateStr)
	})
}

func TestMonthsBetween(t *testing.T) {
	tests := []struct {
		name     string
		d1       Date
		d2       Date
		expected int
	}{
		{
			name:     "same month and year",
			d1:       Date{Month: 1, Year: 2024},
			d2:       Date{Month: 1, Year: 2024},
			expected: 0,
		},
		{
			name:     "consecutive months same year",
			d1:       Date{Month: 1, Year: 2024},
			d2:       Date{Month: 2, Year: 2024},
			expected: 1,
		},
		{
			name:     "6 months apart same year",
			d1:       Date{Month: 1, Year: 2024},
			d2:       Date{Month: 7, Year: 2024},
			expected: 6,
		},
		{
			name:     "december to january next year",
			d1:       Date{Month: 12, Year: 2024},
			d2:       Date{Month: 1, Year: 2025},
			expected: 1,
		},
		{
			name:     "cross year multiple months",
			d1:       Date{Month: 10, Year: 2024},
			d2:       Date{Month: 3, Year: 2025},
			expected: 5,
		},
		{
			name:     "example from question",
			d1:       Date{Month: 12, Year: 2025},
			d2:       Date{Month: 8, Year: 2026},
			expected: 8,
		},
		{
			name:     "exactly 1 year difference same month",
			d1:       Date{Month: 3, Year: 2023},
			d2:       Date{Month: 3, Year: 2024},
			expected: 12,
		},
		{
			name:     "2 years difference",
			d1:       Date{Month: 6, Year: 2020},
			d2:       Date{Month: 6, Year: 2022},
			expected: 24,
		},
		{
			name:     "2 years 5 months difference",
			d1:       Date{Month: 3, Year: 2021},
			d2:       Date{Month: 8, Year: 2023},
			expected: 29,
		},
		{
			name:     "january to december previous year",
			d1:       Date{Month: 1, Year: 2025},
			d2:       Date{Month: 12, Year: 2024},
			expected: 1,
		},
		{
			name:     "large difference",
			d1:       Date{Month: 1, Year: 2000},
			d2:       Date{Month: 12, Year: 2025},
			expected: 311,
		},
		{
			name:     "commutative property - order shouldn't matter",
			d1:       Date{Month: 8, Year: 2026},
			d2:       Date{Month: 12, Year: 2025},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MonthsBetween(tt.d1, tt.d2)
			if result != tt.expected {
				t.Errorf("MonthsBetween(%+v, %+v) = %d, expected %d",
					tt.d1, tt.d2, result, tt.expected)
			}
		})
	}
}
