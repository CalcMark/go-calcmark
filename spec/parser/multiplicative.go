package parser

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/lexer"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
)

// parseMultiplicative parses multiplication, division, modulus, and unit conversions.
// Multiplicative → Exponent ( ('*'|'/'|'%') Exponent )* ('in' UNIT)?
func (p *RecursiveDescentParser) parseMultiplicative() (ast.Node, error) {
	left, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	for p.match(lexer.MULTIPLY, lexer.DIVIDE, lexer.MODULUS) {
		op := p.previous()

		// Special case: Check if DIVIDE might be a rate (e.g., "100 MB/s")
		// Use helper to try parsing as a rate first
		if op.Type == lexer.DIVIDE {
			if rate, ok := p.tryParseRateFromDivision(left, op); ok {
				left = rate
				// Continue to allow further operations: (100 MB/s * 3600)
				// This works because the rate is now 'left' and the loop continues
				continue
			}
		}

		// Not a rate, parse as normal binary operation
		right, err := p.parseExponent()
		if err != nil {
			return nil, err
		}

		// After parsing right operand for division, check if it should be a rate too
		// This handles expressions like (10 req/s / 5 req/s)
		if op.Type == lexer.DIVIDE {
			if rate, ok := p.tryParseRateFromDivision(right, op); ok {
				right = rate
			}
		}

		left = &ast.BinaryOp{
			Operator: string(op.Value),
			Left:     left,
			Right:    right,
			Range:    ast.SpanNodes(left, right),
		}
	}

	// CRITICAL: Check for "downtime" identifier BEFORE checking for PER
	// This prevents "99.9% downtime per month" from being parsed as a rate
	// Must check: percentage/number + "downtime" + "per" + timeunit
	if p.check(lexer.IDENTIFIER) && string(p.peek().Value) == "downtime" {
		downtimeToken := p.peek()
		p.advance() // Consume "downtime"

		// Expect "per" keyword
		if !p.match(lexer.PER) {
			return nil, p.error("expected 'per' after 'downtime'")
		}

		// Parse time period (identifier like "month", "year", etc.)
		if !p.match(lexer.IDENTIFIER) {
			return nil, p.error("expected time period after 'downtime per'")
		}
		timePeriod := p.previous()

		// Validate it's a valid time unit
		if !isTimeUnit(string(timePeriod.Value)) {
			return nil, p.error(fmt.Sprintf("'%s' is not a valid time unit", timePeriod.Value))
		}

		// Create function call: downtime(left, time_period_identifier)
		return &ast.FunctionCall{
			Name: "downtime",
			Arguments: []ast.Node{
				left,
				&ast.Identifier{Name: string(timePeriod.Value)},
			},
			Range: rangeOrFallback(left, downtimeToken),
		}, nil
	}

	// Check for rate with "per" keyword: "5 GB per day"
	// But skip if left is already a RateLiteral (from slash syntax)
	if _, isRate := left.(*ast.RateLiteral); !isRate {
		if p.match(lexer.PER) {
			// When LHS is an Identifier (potentially a rate variable),
			// `<rate> per <period>` desugars to convert_rate(rate, period).
			// The period can be a bare time-unit identifier (fast path),
			// a variable holding a Duration, or a duration literal — runtime
			// extracts the unit. Anything else is rate creation, which still
			// requires a bare time-unit identifier.
			if _, isIdent := left.(*ast.Identifier); isIdent {
				rhs, err := p.parsePerRHS(left.GetRange())
				if err != nil {
					return nil, err
				}
				return &ast.FunctionCall{
					Name:      "convert_rate",
					Arguments: []ast.Node{left, rhs},
					Range:     left.GetRange(),
				}, nil
			}

			// Rate creation: `5 GB per day`. RHS must be a bare time unit.
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected time unit after 'per'")
			}
			timeUnit := string(p.previous().Value)
			if !isTimeUnit(timeUnit) {
				return nil, p.error(fmt.Sprintf("'%s' is not a valid time unit (try %s, or use `<rate> per <unit>` for conversion)", timeUnit, joinedTimeUnits()))
			}

			left = &ast.RateLiteral{
				Amount:     left,
				PerUnit:    timeUnit,
				SourceText: "",
				Range:      p.spanFromNode(left),
			}
		}
	}

	// Check for "over" keyword: "100 MB/s over 1 day"
	// Natural syntax for accumulate(rate, time_period)
	if p.check(lexer.OVER) {
		overToken := p.peek()
		p.advance() // consume OVER

		// Parse duration/time period
		duration, err := p.parseExponent()
		if err != nil {
			return nil, err
		}

		// Create function call: accumulate(left, duration)
		return &ast.FunctionCall{
			Name:      "accumulate",
			Arguments: []ast.Node{left, duration},
			Range:     rangeOrFallback(left, overToken),
		}, nil
	}

	// Check for "per" after a rate (conversion context)
	// Example: "(100 MB/day) per second" — converts existing rate.
	// Same NL contract as the Identifier-LHS branch above: the period
	// can be a bare time-unit identifier, a Duration variable, or a
	// duration literal.
	if _, isRate := left.(*ast.RateLiteral); isRate {
		if p.match(lexer.PER) {
			rhs, err := p.parsePerRHS(left.GetRange())
			if err != nil {
				return nil, err
			}
			return &ast.FunctionCall{
				Name:      "convert_rate",
				Arguments: []ast.Node{left, rhs},
				Range:     left.GetRange(),
			}, nil
		}
	}

	// Check for "at" keyword: "10 TB at 2 TB per disk"
	// Natural syntax for capacity(demand, capacity_per_unit, unit, buffer?)
	// Returns a Quantity with the specified unit
	if p.match(lexer.AT) {
		// Parse capacity expression - use parseExponent() to avoid consuming "per"
		// The "per" keyword belongs to our syntax, not a rate expression
		capacityExpr, err := p.parseCapacityValue()
		if err != nil {
			return nil, err
		}

		// Require "per" keyword OR "/" operator
		// Supports both: "10 TB at 2 TB per disk" and "10 TB at 2 TB/disk"
		var unitName string
		if p.match(lexer.PER) {
			// "per disk" syntax
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected unit name after 'per' (e.g., 'disk', 'server', 'crate')")
			}
			unitName = string(p.previous().Value)
		} else if p.match(lexer.DIVIDE) {
			// "/disk" syntax
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected unit name after '/' (e.g., 'disk', 'server', 'crate')")
			}
			unitName = string(p.previous().Value)
		} else {
			return nil, p.error("expected 'per' or '/' after capacity in 'X at Y per UNIT' syntax")
		}
		unitNode := &ast.Identifier{
			Name:  unitName,
			Range: left.GetRange(),
		}

		// Check for optional "with N% buffer"
		var args []ast.Node
		if p.match(lexer.WITH) {
			// Parse buffer percentage
			bufferExpr, err := p.parseExponent()
			if err != nil {
				return nil, err
			}

			// Require "buffer" keyword after the percentage
			if !p.match(lexer.IDENTIFIER) || p.previous().Value != "buffer" {
				return nil, p.error("expected 'buffer' after percentage in 'with N% buffer' syntax")
			}

			args = []ast.Node{left, capacityExpr, unitNode, bufferExpr}
		} else {
			args = []ast.Node{left, capacityExpr, unitNode}
		}

		// Create function call: capacity(demand, capacity_per_unit, unit, buffer?)
		return &ast.FunctionCall{
			Name:      "capacity",
			Arguments: args,
			Range:     left.GetRange(),
		}, nil
	}

	// Check for unit conversion: "10 meters in feet" or "10 feet in nautical miles"
	// Also handles rate unit conversion: "10 m/s in inch/s"
	// Also handles currency conversion: "100 USD in EUR"
	if p.match(lexer.IN) {
		if !p.match(lexer.IDENTIFIER) && !p.match(lexer.CURRENCY_CODE) {
			return nil, p.error("expected unit name or currency code after 'in'")
		}
		targetUnit := p.previous()
		targetUnitName := string(targetUnit.Value)

		// Check for multi-word target unit: "in nautical miles", "in meters per second squared"
		if p.check(lexer.IDENTIFIER) {
			nextWord := string(p.peek().Value)
			if multiWordUnit := units.IsMultiWordUnit(targetUnitName, nextWord); multiWordUnit != "" {
				p.advance() // Consume the second word
				targetUnitName = multiWordUnit

				// Check for 3rd word: "in meters per second"
				if p.check(lexer.IDENTIFIER) {
					thirdWord := string(p.peek().Value)
					if multiWordUnit3 := units.IsMultiWordUnit(targetUnitName, thirdWord); multiWordUnit3 != "" {
						p.advance() // Consume the third word
						targetUnitName = multiWordUnit3

						// Check for 4th word: "in meters per second squared"
						if p.check(lexer.IDENTIFIER) {
							fourthWord := string(p.peek().Value)
							if multiWordUnit4 := units.IsMultiWordUnit(targetUnitName, fourthWord); multiWordUnit4 != "" {
								p.advance() // Consume the fourth word
								targetUnitName = multiWordUnit4
							}
						}
					}
				}
			}
		}

		// Check for hyphenated target unit: "in pound-force", "in newton-seconds"
		if p.check(lexer.MINUS) {
			savedPos := p.current
			p.advance() // Consume MINUS
			if p.check(lexer.IDENTIFIER) {
				nextWord := string(p.peek().Value)
				if hyphenUnit := units.IsHyphenatedUnit(targetUnitName, nextWord); hyphenUnit != "" {
					p.advance() // Consume the second word
					targetUnitName = hyphenUnit

					// Check for triple-hyphenated: "in pound-force-seconds"
					if p.check(lexer.MINUS) {
						savedPos3 := p.current
						p.advance() // Consume second MINUS
						if p.check(lexer.IDENTIFIER) {
							thirdWord := string(p.peek().Value)
							if tripleUnit := units.IsHyphenatedTripleUnit(targetUnitName, thirdWord); tripleUnit != "" {
								p.advance() // Consume the third word
								targetUnitName = tripleUnit
							} else {
								p.current = savedPos3
							}
						} else {
							p.current = savedPos3
						}
					}
				} else {
					p.current = savedPos
				}
			} else {
				p.current = savedPos
			}
		}

		// Check for rate target unit: "in inch/s" or "in inch per second"
		var targetTimeUnit string
		if p.match(lexer.DIVIDE) {
			// Rate syntax: "in inch/s"
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected time unit after '/' in rate conversion")
			}
			timeUnit := string(p.previous().Value)
			if !isTimeUnit(timeUnit) {
				return nil, p.error(fmt.Sprintf("'%s' is not a valid time unit for rate conversion", timeUnit))
			}
			targetTimeUnit = timeUnit
		} else if p.match(lexer.PER) {
			// Natural syntax: "in inch per second"
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected time unit after 'per' in rate conversion")
			}
			timeUnit := string(p.previous().Value)
			if !isTimeUnit(timeUnit) {
				return nil, p.error(fmt.Sprintf("'%s' is not a valid time unit for rate conversion", timeUnit))
			}
			targetTimeUnit = timeUnit
		}

		// Resolve time unit abbreviations for duration-only conversions (e.g., "ms" → "millisecond")
		// Only normalize when there's no rate time unit (targetTimeUnit == ""),
		// to avoid converting quantity units like "m" (meters) to "minute".
		if targetTimeUnit == "" {
			if normalized := types.NormalizeTimeUnit(targetUnitName); types.IsValidDurationUnit(normalized) && !types.IsValidDurationUnit(targetUnitName) {
				targetUnitName = normalized
			}
		}

		return &ast.UnitConversion{
			Quantity:       left,
			TargetUnit:     targetUnitName,
			TargetTimeUnit: targetTimeUnit,
			Range:          left.GetRange(),
		}, nil
	}

	return left, nil
}

// parsePerRHS parses the right-hand side of the NL `<rate> per <period>`
// conversion form. The fast path is a bare time-unit identifier
// (`day`, `quarter`, `s`) — returned as an Identifier node so the
// interpreter treats the name as the target unit.
//
// Otherwise the RHS is parsed as a full expression (parseExponent) so a
// variable holding a Duration or a duration literal like `1 day` can
// stand in for the period. The runtime extracts the unit from the
// evaluated value; the numerical magnitude is ignored.
//
// fallbackRange is used as the Range for the synthesized Identifier
// when we take the fast path.
func (p *RecursiveDescentParser) parsePerRHS(fallbackRange *ast.Range) (ast.Node, error) {
	// Fast path: bare IDENTIFIER that's a recognized time unit.
	if p.check(lexer.IDENTIFIER) {
		name := string(p.peek().Value)
		if isTimeUnit(name) {
			p.advance()
			return &ast.Identifier{
				Name:  name,
				Range: fallbackRange,
			}, nil
		}
	}
	// Defer to runtime — RHS can be a Duration variable, a duration
	// literal, or an unknown identifier that the interpreter will
	// report cleanly.
	return p.parseExponent()
}

// joinedTimeUnits returns the comma-separated canonical time-unit list
// used in parse-time diagnostics so users see what's accepted after
// `per`. Kept here (rather than in the identifiers package) so the
// parser package stays free of an interpreter-layer dependency.
func joinedTimeUnits() string {
	return "second, minute, hour, day, week, month, quarter, year"
}
