package units

import "testing"

func TestResolveUnit_DefaultConfig(t *testing.T) {
	// With nil config, nothing changes
	tests := []struct {
		input string
		want  string
	}{
		{"oz", "oz"},
		{"gallon", "gallon"},
		{"ton", "ton"},
		{"troy oz", "troy oz"},
		{"imp gal", "imp gal"},
	}
	for _, tt := range tests {
		got := ResolveUnit(tt.input, nil)
		if got != tt.want {
			t.Errorf("ResolveUnit(%q, nil) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveUnit_ImperialVolume(t *testing.T) {
	mc := &MeasurementConfig{Volume: "imperial", Mass: "standard", Ton: "short"}
	tests := []struct {
		input string
		want  string
	}{
		{"gallon", "imperial gallon"},
		{"gallons", "imperial gallon"},
		{"gal", "imperial gallon"},
		{"pint", "imperial pint"},
		{"pt", "imperial pint"},
		{"quart", "imperial quart"},
		{"qt", "imperial quart"},
		{"cup", "imperial cup"},
		{"fl oz", "imperial fluid ounce"},
		// Non-ambiguous units pass through
		{"meter", "meter"},
		{"kg", "kg"},
		// Already qualified units pass through
		{"us gal", "us gal"},
		{"imp gal", "imp gal"},
	}
	for _, tt := range tests {
		got := ResolveUnit(tt.input, mc)
		if got != tt.want {
			t.Errorf("ResolveUnit(%q, imperial) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveUnit_TroyMass(t *testing.T) {
	mc := &MeasurementConfig{Volume: "us", Mass: "troy", Ton: "short"}
	tests := []struct {
		input string
		want  string
	}{
		{"oz", "troy ounce"},
		{"ounce", "troy ounce"},
		{"lb", "troy pound"},
		{"pound", "troy pound"},
		// Volume unaffected
		{"gallon", "gallon"},
		// Already qualified
		{"us oz", "us oz"},
		{"troy oz", "troy oz"},
	}
	for _, tt := range tests {
		got := ResolveUnit(tt.input, mc)
		if got != tt.want {
			t.Errorf("ResolveUnit(%q, troy) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveUnit_LongTon(t *testing.T) {
	mc := &MeasurementConfig{Volume: "us", Mass: "standard", Ton: "long"}
	tests := []struct {
		input string
		want  string
	}{
		{"ton", "long ton"},
		{"tons", "long ton"},
		// Other axes unaffected
		{"oz", "oz"},
		{"gallon", "gallon"},
	}
	for _, tt := range tests {
		got := ResolveUnit(tt.input, mc)
		if got != tt.want {
			t.Errorf("ResolveUnit(%q, long ton) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveUnit_MetricTon(t *testing.T) {
	mc := &MeasurementConfig{Volume: "us", Mass: "standard", Ton: "metric"}
	got := ResolveUnit("ton", mc)
	if got != "metric ton" {
		t.Errorf("ResolveUnit(ton, metric) = %q, want %q", got, "metric ton")
	}
}

func TestIsAmbiguousUnit(t *testing.T) {
	ambiguous := []string{"oz", "ounce", "lb", "pound", "gallon", "gal", "pint", "pt", "cup", "ton", "fl oz"}
	for _, u := range ambiguous {
		if !IsAmbiguousUnit(u) {
			t.Errorf("IsAmbiguousUnit(%q) = false, want true", u)
		}
	}
	notAmbiguous := []string{"meter", "kg", "celsius", "troy oz", "imp gal", "us oz", "short ton"}
	for _, u := range notAmbiguous {
		if IsAmbiguousUnit(u) {
			t.Errorf("IsAmbiguousUnit(%q) = true, want false", u)
		}
	}
}

func TestConventionPrefix_Default(t *testing.T) {
	mc := DefaultMeasurement()
	tests := []struct {
		input string
		want  string
	}{
		{"oz", "us"},
		{"gallon", "us"},
		{"ton", "short"},
		{"meter", ""},   // not ambiguous
		{"troy oz", ""}, // already qualified
		{"imp gal", ""}, // already qualified
	}
	for _, tt := range tests {
		got := ConventionPrefix(tt.input, mc)
		if got != tt.want {
			t.Errorf("ConventionPrefix(%q, default) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConventionPrefix_NonDefault(t *testing.T) {
	mc := &MeasurementConfig{Volume: "imperial", Mass: "troy", Ton: "long"}
	tests := []struct {
		input string
		want  string
	}{
		{"oz", "troy"},
		{"gallon", "imp"},
		{"ton", "long"},
	}
	for _, tt := range tests {
		got := ConventionPrefix(tt.input, mc)
		if got != tt.want {
			t.Errorf("ConventionPrefix(%q, non-default) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
