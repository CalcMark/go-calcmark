package lexer

// DateKeywords maps date keyword strings to token types
// Performance: O(1) lookup via map
var DateKeywords = map[string]TokenType{
	// Basic keywords
	"today":     DATE_TODAY,
	"tomorrow":  DATE_TOMORROW,
	"yesterday": DATE_YESTERDAY,

	// Time keywords
	"now": DATE_TODAY, // "now" tokenizes like "today" but evaluator preserves time

	// Duration modifier
	"ago": AGO,

	// Bare weekday names (shorthand for "this <weekday>")
	"monday":    DATE_WEEKDAY,
	"tuesday":   DATE_WEEKDAY,
	"wednesday": DATE_WEEKDAY,
	"thursday":  DATE_WEEKDAY,
	"friday":    DATE_WEEKDAY,
	"saturday":  DATE_WEEKDAY,
	"sunday":    DATE_WEEKDAY,

	// NOTE: bare period abbreviations (CY / FY / CQ / FQ) are NOT
	// registered here. They collide with the notation parser
	// (Q1-Q4, FQ1-FQ4, FY2026, CY26) because the word scanner
	// stops at the first non-letter, consuming `FQ` out of `FQ1`.
	// Users reach the same semantics via the relative forms below
	// (`this CY`, `next FY`, `last FQ`, etc.).
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

	// Start/end-of modifiers
	"start of": START_OF,
	"end of":   END_OF,

	// v2.0 Period operators
	"length of": LENGTH_OF, // `length of <Period>` → Duration
	"days in":   DAYS_IN,   // `days in <Period>` → Number

	// Bare-form fiscal period aliases. `end of fiscal quarter` reads
	// naturally in English; users already configure fiscal_year_starts
	// in frontmatter, so the implied "this" is unambiguous. Single-word
	// forms (`quarter`, `year`, `month`, `week`) stay as identifiers
	// because they conflict with common variable names; the multi-word
	// fiscal phrases use a space and don't collide with snake_case
	// variable conventions.
	"fiscal quarter": DATE_THIS_FISCAL_QUARTER,
	"fiscal year":    DATE_THIS_FISCAL_YEAR,

	// Calendar quarters
	"this quarter": DATE_THIS_QUARTER,
	"next quarter": DATE_NEXT_QUARTER,
	"last quarter": DATE_LAST_QUARTER,

	// Period abbreviations with relative prefixes.
	// Calendar year / fiscal year / calendar quarter / fiscal quarter.
	"this cy": DATE_THIS_YEAR,
	"next cy": DATE_NEXT_YEAR,
	"last cy": DATE_LAST_YEAR,

	"this fy": DATE_THIS_FISCAL_YEAR,
	"next fy": DATE_NEXT_FISCAL_YEAR,
	"last fy": DATE_LAST_FISCAL_YEAR,

	"this cq": DATE_THIS_QUARTER,
	"next cq": DATE_NEXT_QUARTER,
	"last cq": DATE_LAST_QUARTER,

	"this fq": DATE_THIS_FISCAL_QUARTER,
	"next fq": DATE_NEXT_FISCAL_QUARTER,
	"last fq": DATE_LAST_FISCAL_QUARTER,

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

	// This month name (full + abbreviation)
	"this january": DATE_THIS_MONTH_NAME, "this jan": DATE_THIS_MONTH_NAME,
	"this february": DATE_THIS_MONTH_NAME, "this feb": DATE_THIS_MONTH_NAME,
	"this march": DATE_THIS_MONTH_NAME, "this mar": DATE_THIS_MONTH_NAME,
	"this april": DATE_THIS_MONTH_NAME, "this apr": DATE_THIS_MONTH_NAME,
	"this may":  DATE_THIS_MONTH_NAME,
	"this june": DATE_THIS_MONTH_NAME, "this jun": DATE_THIS_MONTH_NAME,
	"this july": DATE_THIS_MONTH_NAME, "this jul": DATE_THIS_MONTH_NAME,
	"this august": DATE_THIS_MONTH_NAME, "this aug": DATE_THIS_MONTH_NAME,
	"this september": DATE_THIS_MONTH_NAME, "this sep": DATE_THIS_MONTH_NAME, "this sept": DATE_THIS_MONTH_NAME,
	"this october": DATE_THIS_MONTH_NAME, "this oct": DATE_THIS_MONTH_NAME,
	"this november": DATE_THIS_MONTH_NAME, "this nov": DATE_THIS_MONTH_NAME,
	"this december": DATE_THIS_MONTH_NAME, "this dec": DATE_THIS_MONTH_NAME,

	// Next month name
	"next january": DATE_NEXT_MONTH_NAME, "next jan": DATE_NEXT_MONTH_NAME,
	"next february": DATE_NEXT_MONTH_NAME, "next feb": DATE_NEXT_MONTH_NAME,
	"next march": DATE_NEXT_MONTH_NAME, "next mar": DATE_NEXT_MONTH_NAME,
	"next april": DATE_NEXT_MONTH_NAME, "next apr": DATE_NEXT_MONTH_NAME,
	"next may":  DATE_NEXT_MONTH_NAME,
	"next june": DATE_NEXT_MONTH_NAME, "next jun": DATE_NEXT_MONTH_NAME,
	"next july": DATE_NEXT_MONTH_NAME, "next jul": DATE_NEXT_MONTH_NAME,
	"next august": DATE_NEXT_MONTH_NAME, "next aug": DATE_NEXT_MONTH_NAME,
	"next september": DATE_NEXT_MONTH_NAME, "next sep": DATE_NEXT_MONTH_NAME, "next sept": DATE_NEXT_MONTH_NAME,
	"next october": DATE_NEXT_MONTH_NAME, "next oct": DATE_NEXT_MONTH_NAME,
	"next november": DATE_NEXT_MONTH_NAME, "next nov": DATE_NEXT_MONTH_NAME,
	"next december": DATE_NEXT_MONTH_NAME, "next dec": DATE_NEXT_MONTH_NAME,

	// Last month name
	"last january": DATE_LAST_MONTH_NAME, "last jan": DATE_LAST_MONTH_NAME,
	"last february": DATE_LAST_MONTH_NAME, "last feb": DATE_LAST_MONTH_NAME,
	"last march": DATE_LAST_MONTH_NAME, "last mar": DATE_LAST_MONTH_NAME,
	"last april": DATE_LAST_MONTH_NAME, "last apr": DATE_LAST_MONTH_NAME,
	"last may":  DATE_LAST_MONTH_NAME,
	"last june": DATE_LAST_MONTH_NAME, "last jun": DATE_LAST_MONTH_NAME,
	"last july": DATE_LAST_MONTH_NAME, "last jul": DATE_LAST_MONTH_NAME,
	"last august": DATE_LAST_MONTH_NAME, "last aug": DATE_LAST_MONTH_NAME,
	"last september": DATE_LAST_MONTH_NAME, "last sep": DATE_LAST_MONTH_NAME, "last sept": DATE_LAST_MONTH_NAME,
	"last october": DATE_LAST_MONTH_NAME, "last oct": DATE_LAST_MONTH_NAME,
	"last november": DATE_LAST_MONTH_NAME, "last nov": DATE_LAST_MONTH_NAME,
	"last december": DATE_LAST_MONTH_NAME, "last dec": DATE_LAST_MONTH_NAME,
}

// ThreeWordDateKeywords maps three-word fiscal phrases to token types.
// Checked after single-word and two-word lookups fail.
var ThreeWordDateKeywords = map[string]TokenType{
	"this fiscal quarter": DATE_THIS_FISCAL_QUARTER,
	"next fiscal quarter": DATE_NEXT_FISCAL_QUARTER,
	"last fiscal quarter": DATE_LAST_FISCAL_QUARTER,
	"this fiscal year":    DATE_THIS_FISCAL_YEAR,
	"next fiscal year":    DATE_NEXT_FISCAL_YEAR,
	"last fiscal year":    DATE_LAST_FISCAL_YEAR,
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
