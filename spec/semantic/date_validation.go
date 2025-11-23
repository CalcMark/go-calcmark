package semantic

import (
	"fmt"
	"strconv"
	"time"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// checkDateLiteral validates date literals
// USER REQUIREMENT: Validate Feb 30, leap years, day ranges
func (c *Checker) checkDateLiteral(node *ast.DateLiteral) {
	if node == nil {
		return
	}

	// Parse month
	monthNum := monthNameToNumber(node.Month)
	if monthNum == 0 {
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidMonth,
			Message:  "invalid month",
			Detailed: fmt.Sprintf("'%s' is not a valid month name", node.Month),
			Range:    node.Range,
		})
		return
	}

	// Parse day
	day, err := strconv.Atoi(node.Day)
	if err != nil || day < 1 || day > 31 {
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidDay,
			Message:  "invalid day",
			Detailed: fmt.Sprintf("Day must be between 1 and 31, got %s", node.Day),
			Range:    node.Range,
		})
		return
	}

	// Parse year (if present)
	year := time.Now().Year() // Default to current year
	if node.Year != nil {
		year, err = strconv.Atoi(*node.Year)
		if err != nil || year < 1900 || year > 2100 {
			c.addDiagnostic(Diagnostic{
				Severity: Error,
				Code:     DiagInvalidYear,
				Message:  "invalid year",
				Detailed: fmt.Sprintf("Year must be between 1900 and 2100, got %s", *node.Year),
				Range:    node.Range,
			})
			return
		}
	}

	// Validate day for specific month (USER REQUIREMENT)
	maxDays := daysInMonth(monthNum, year)
	if day > maxDays {
		monthName := node.Month
		var detailed string

		if monthNum == 2 && maxDays == 29 {
			// Leap year February
			detailed = fmt.Sprintf("%s only has %d days in %d (leap year)", monthName, maxDays, year)
		} else if monthNum == 2 {
			// Non-leap year February
			detailed = fmt.Sprintf("%s only has %d days in %d (not a leap year)", monthName, maxDays, year)
		} else {
			// Other months
			detailed = fmt.Sprintf("%s only has %d days", monthName, maxDays)
		}

		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidDate,
			Message:  "invalid date",
			Detailed: detailed,
			Range:    node.Range,
		})
	}
}

// Helper functions for date validation

func monthNameToNumber(name string) int {
	months := map[string]int{
		"January": 1, "February": 2, "March": 3, "April": 4,
		"May": 5, "June": 6, "July": 7, "August": 8,
		"September": 9, "October": 10, "November": 11, "December": 12,
	}
	return months[name]
}

func daysInMonth(month, year int) int {
	if month < 1 || month > 12 {
		return 0
	}

	days := []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	// Leap year handling for February (USER REQUIREMENT)
	if month == 2 && isLeapYear(year) {
		return 29
	}

	return days[month]
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
