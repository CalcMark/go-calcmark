// Package identifiers provides the canonical set of named values accepted
// by CalcMark functions. This is the single source of truth for network scopes,
// network types, storage types, and compression types.
//
// All consumers (spec/semantic, spec/features, impl/interpreter) derive their
// identifier lists from this package. This eliminates drift between layers.
package identifiers

import "strings"

// NetworkScopes lists valid scope identifiers for rtt() and transfer_time().
// Ordered by most commonly used first — first element is the representative
// example shown in the TUI compact footer.
var NetworkScopes = []string{"local", "regional", "continental", "global"}

// NetworkTypes lists valid network type identifiers for throughput() and transfer_time().
var NetworkTypes = []string{"gigabit", "ten_gig", "hundred_gig", "wifi", "four_g", "five_g"}

// StorageTypes lists valid storage type identifiers for read() and seek().
var StorageTypes = []string{"ssd", "nvme", "pcie_ssd", "hdd"}

// StorageAliases maps alternative storage names to their canonical name.
// The interpreter accepts both the canonical name and any alias.
var StorageAliases = map[string]string{"sata_ssd": "ssd"}

// CompressionTypes lists valid compression type identifiers for compress().
var CompressionTypes = []string{"gzip", "zstd", "lz4", "snappy", "bzip2", "none"}

// TimeUnits lists valid time unit identifiers accepted by convert_rate() and
// other rate/duration-manipulating functions. Ordered from smallest to largest
// so "second" is the representative example.
var TimeUnits = []string{"nanosecond", "microsecond", "millisecond", "second", "minute", "hour", "day", "week", "month", "year"}

// AllStorageNames returns all valid storage names including aliases.
// Used by interpreter validation and error messages.
func AllStorageNames() []string {
	names := make([]string, 0, len(StorageTypes)+len(StorageAliases))
	names = append(names, StorageTypes...)
	for alias := range StorageAliases {
		names = append(names, alias)
	}
	return names
}

// JoinNames formats a name list for error messages: "ssd, nvme, pcie_ssd, hdd".
func JoinNames(names []string) string {
	return strings.Join(names, ", ")
}
