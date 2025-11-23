package types

// Type is the interface that all CalcMark value types implement.
// All types can be converted to a human-readable string representation.
type Type interface {
	// String returns a human-readable representation of the value.
	// This is used for display purposes and debugging.
	String() string
}
