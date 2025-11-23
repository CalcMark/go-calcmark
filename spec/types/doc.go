// Package types defines the CalcMark type system.
//
// CalcMark supports several value types for calculations and data representation.
// All types implement the Type interface which provides string representation.
//
// # Core Types
//
//   - Number: Arbitrary-precision decimal numbers
//   - Currency: Monetary values with currency symbol/code
//   - Quantity: Physical quantities with units (e.g., "5 kg", "10 meters")
//   - Date: Calendar dates with date arithmetic
//   - Time: Time of day with timezone support
//   - Duration: Time durations (e.g., "5 days", "3 hours")
//   - Boolean: True/false values
//
// # Number Type
//
// Numbers use arbitrary-precision decimals via shopspring/decimal:
//
//	num := types.NewNumber(decimal.NewFromFloat(3.14159))
//	fmt.Println(num.String()) // "3.14159"
//
// Supports thousands separators:
//
//	1,000
//	1_000_000
//
// # Currency Type
//
// Currency values combine a decimal amount with a symbol or ISO code:
//
//	usd := types.NewCurrency(decimal.NewFromInt(100), "$")
//	eur := types.NewCurrencyFromCode(decimal.NewFromInt(50), "EUR")
//
// Supported symbols: $, €, £, ¥
// Supports ISO 4217 currency codes (e.g., USD, EUR, GBP, JPY)
//
// # Quantity Type
//
// Physical quantities with units:
//
//	mass := types.NewQuantity(decimal.NewFromInt(5), "kg")
//	length := types.NewQuantity(decimal.NewFromInt(10), "meters")
//
// Supports SI and US customary units:
//   - Length: meters, feet, kilometers, miles, etc.
//   - Mass: kilograms, pounds, grams, etc.
//   - Volume: liters, gallons, cups, etc.
//
// # Date Type
//
// Calendar dates with arithmetic operations:
//
//	today := types.NewDateFromTime(time.Now())
//	nextWeek := today.AddDays(7)
//	daysBetween := today.DaysBetween(nextWeek) // 7
//
// # Time Type
//
// Time of day with timezone support:
//
//	morning := types.NewTime(9, 30, 0, false, 0) // 9:30 AM UTC
//	evening := types.NewTime(6, 0, 0, true, -300) // 6:00 PM EST
//
// # Duration Type
//
// Time durations for date arithmetic:
//
//	week := types.NewDuration(7, types.DurationUnitDay)
//	hour := types.NewDuration(1, types.DurationUnitHour)
//
// # Boolean Type
//
// Simple true/false values:
//
//	t := types.NewBoolean(true)
//	f := types.NewBoolean(false)
//
// # Type Conversions
//
// The type system maintains type safety while allowing sensible conversions:
//   - Currency + Number → Currency (preserves currency)
//   - Currency + Currency (different) → Number (drops units)
//   - Quantity + Quantity (compatible) → Quantity (first unit wins)
//
// # Performance
//
// All type operations are designed for speed and use efficient decimal
// arithmetic. String formatting is optimized for display/debugging.
package types
