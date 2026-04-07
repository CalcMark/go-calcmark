package lexer

// DateKeywords maps date keyword strings to token types
// Performance: O(1) lookup via map
var DateKeywords = map[string]TokenType{
	// Basic keywords
	"today":     DATE_TODAY,
	"tomorrow":  DATE_TOMORROW,
	"yesterday": DATE_YESTERDAY,

	// Bare weekday names (shorthand for "this <weekday>")
	"monday":    DATE_WEEKDAY,
	"tuesday":   DATE_WEEKDAY,
	"wednesday": DATE_WEEKDAY,
	"thursday":  DATE_WEEKDAY,
	"friday":    DATE_WEEKDAY,
	"saturday":  DATE_WEEKDAY,
	"sunday":    DATE_WEEKDAY,
}

// RelativeDateKeywords maps multi-word relative date keywords to token types
// These are checked as phrases (e.g., "this week")
// Performance: O(1) lookup via map
var RelativeDateKeywords = map[string]TokenType{
	// This
	"this week":  DATE_THIS_WEEK,
	"this month": DATE_THIS_MONTH,
	"this year":  DATE_THIS_YEAR,

	// Next
	"next week":  DATE_NEXT_WEEK,
	"next month": DATE_NEXT_MONTH,
	"next year":  DATE_NEXT_YEAR,

	// Last
	"last week":  DATE_LAST_WEEK,
	"last month": DATE_LAST_MONTH,
	"last year":  DATE_LAST_YEAR,

	// This weekday
	"this monday":    DATE_THIS_WEEKDAY,
	"this tuesday":   DATE_THIS_WEEKDAY,
	"this wednesday": DATE_THIS_WEEKDAY,
	"this thursday":  DATE_THIS_WEEKDAY,
	"this friday":    DATE_THIS_WEEKDAY,
	"this saturday":  DATE_THIS_WEEKDAY,
	"this sunday":    DATE_THIS_WEEKDAY,

	// Next weekday
	"next monday":    DATE_NEXT_WEEKDAY,
	"next tuesday":   DATE_NEXT_WEEKDAY,
	"next wednesday": DATE_NEXT_WEEKDAY,
	"next thursday":  DATE_NEXT_WEEKDAY,
	"next friday":    DATE_NEXT_WEEKDAY,
	"next saturday":  DATE_NEXT_WEEKDAY,
	"next sunday":    DATE_NEXT_WEEKDAY,

	// Last weekday
	"last monday":    DATE_LAST_WEEKDAY,
	"last tuesday":   DATE_LAST_WEEKDAY,
	"last wednesday": DATE_LAST_WEEKDAY,
	"last thursday":  DATE_LAST_WEEKDAY,
	"last friday":    DATE_LAST_WEEKDAY,
	"last saturday":  DATE_LAST_WEEKDAY,
	"last sunday":    DATE_LAST_WEEKDAY,
}

// MonthNames maps month abbreviations and full names to canonical month names
// Performance: O(1) lookup via map
var MonthNames = map[string]string{
	// January
	"jan":     "January",
	"january": "January",

	// February
	"feb":      "February",
	"february": "February",

	// March
	"mar":   "March",
	"march": "March",

	// April
	"apr":   "April",
	"april": "April",

	// May
	"may": "May",

	// June
	"jun":  "June",
	"june": "June",

	// July
	"jul":  "July",
	"july": "July",

	// August
	"aug":    "August",
	"august": "August",

	// September
	"sep":       "September",
	"sept":      "September",
	"september": "September",

	// October
	"oct":     "October",
	"october": "October",

	// November
	"nov":      "November",
	"november": "November",

	// December
	"dec":      "December",
	"december": "December",
}

// TimeUnits maps time unit keywords to canonical forms
// Performance: O(1) lookup via map
var TimeUnits = map[string]string{
	// Milliseconds
	"millisecond":  "millisecond",
	"milliseconds": "millisecond",
	"ms":           "millisecond",

	// Seconds
	"second":  "second",
	"seconds": "second",
	"sec":     "second",
	"secs":    "second",

	// Minutes
	"minute":  "minute",
	"minutes": "minute",
	"min":     "minute",
	"mins":    "minute",

	// Hours
	"hour":  "hour",
	"hours": "hour",
	"hr":    "hour",
	"hrs":   "hour",

	// Days
	"day":  "day",
	"days": "day",

	// Weeks
	"week":  "week",
	"weeks": "week",

	// Months
	"month":  "month",
	"months": "month",

	// Years
	"year":  "year",
	"years": "year",
	"yr":    "year",
	"yrs":   "year",
}
