package document

import "testing"

func TestRegistry_LookupExchange(t *testing.T) {
	rk, ok := LookupKey("exchange")
	if !ok {
		t.Fatal("expected to find 'exchange' in registry")
	}
	if rk.Type != FrontmatterKeyMapStringDecimal {
		t.Errorf("expected exchange to be MapStringDecimal, got %v", rk.Type)
	}
}

func TestRegistry_LookupGlobals(t *testing.T) {
	rk, ok := LookupKey("globals")
	if !ok {
		t.Fatal("expected to find 'globals' in registry")
	}
	if rk.Type != FrontmatterKeyMapStringString {
		t.Errorf("expected globals to be MapStringString, got %v", rk.Type)
	}
}

func TestRegistry_LookupConvertToEnum(t *testing.T) {
	rk, ok := LookupKey("convert_to")
	if !ok {
		t.Fatal("expected to find 'convert_to' in registry")
	}
	if rk.Type != FrontmatterKeyEnumString {
		t.Errorf("expected convert_to to be EnumString, got %v", rk.Type)
	}
	want := []string{"si", "imperial"}
	if len(rk.EnumValues) != len(want) {
		t.Fatalf("expected %d enum values, got %v", len(want), rk.EnumValues)
	}
	for i, v := range want {
		if rk.EnumValues[i] != v {
			t.Errorf("EnumValues[%d]: want %q got %q", i, v, rk.EnumValues[i])
		}
	}
}

func TestRegistry_LookupScale(t *testing.T) {
	rk, ok := LookupKey("scale")
	if !ok {
		t.Fatal("expected to find 'scale' in registry")
	}
	if rk.Type != FrontmatterKeyStruct {
		t.Errorf("expected scale to be Struct, got %v", rk.Type)
	}
}

func TestRegistry_LookupMeasurement(t *testing.T) {
	rk, ok := LookupKey("measurement")
	if !ok {
		t.Fatal("expected to find 'measurement' in registry")
	}
	if rk.Type != FrontmatterKeyStruct {
		t.Errorf("expected measurement to be Struct, got %v", rk.Type)
	}
}

func TestRegistry_LookupFiscalYearStarts(t *testing.T) {
	rk, ok := LookupKey("fiscal_year_starts")
	if !ok {
		t.Fatal("expected to find 'fiscal_year_starts' in registry")
	}
	if rk.Type != FrontmatterKeyStruct {
		t.Errorf("expected fiscal_year_starts to be Struct, got %v", rk.Type)
	}
}

func TestRegistry_AllEntriesHaveNameAndDoc(t *testing.T) {
	for _, entry := range Registry {
		if entry.Name == "" {
			t.Errorf("registry entry has empty Name: %+v", entry)
		}
		if entry.Doc == "" {
			t.Errorf("registry entry %q has empty Doc", entry.Name)
		}
	}
}

func TestRegistry_EnumStringEntriesHaveValues(t *testing.T) {
	for _, entry := range Registry {
		if entry.Type == FrontmatterKeyEnumString {
			if len(entry.EnumValues) == 0 {
				t.Errorf("EnumString entry %q has no EnumValues", entry.Name)
			}
		}
	}
}

func TestRegistry_NonEnumEntriesHaveEmptyEnumValues(t *testing.T) {
	for _, entry := range Registry {
		if entry.Type != FrontmatterKeyEnumString {
			if len(entry.EnumValues) != 0 {
				t.Errorf("non-EnumString entry %q has unexpected EnumValues: %v",
					entry.Name, entry.EnumValues)
			}
		}
	}
}

func TestRegistry_LookupCaseSensitive(t *testing.T) {
	if _, ok := LookupKey("Exchange"); ok {
		t.Error("expected LookupKey to be case-sensitive; 'Exchange' should not match")
	}
}

func TestRegistry_LookupNonexistent(t *testing.T) {
	if _, ok := LookupKey("nonexistent_key"); ok {
		t.Error("expected LookupKey to return false for unknown key")
	}
}

func TestRegistry_IsRegisteredKey(t *testing.T) {
	known := []string{"exchange", "globals", "scale", "convert_to", "measurement", "fiscal_year_starts"}
	for _, k := range known {
		if !IsRegisteredKey(k) {
			t.Errorf("expected %q to be a registered key", k)
		}
	}
	if IsRegisteredKey("title") {
		t.Error("expected 'title' to NOT be a registered key")
	}
	if IsRegisteredKey("xyzzy") {
		t.Error("expected 'xyzzy' to NOT be a registered key")
	}
	if IsRegisteredKey("Exchange") {
		t.Error("expected 'Exchange' (capitalized) to NOT be a registered key")
	}
}

func TestRegistry_HasAllSixKnownKeys(t *testing.T) {
	if len(Registry) != 6 {
		t.Errorf("expected 6 registered keys, got %d", len(Registry))
	}
}
