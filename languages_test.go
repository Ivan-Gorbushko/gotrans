package gotrans

import (
	"testing"
)

func TestParseLocale(t *testing.T) {
	tests := []struct {
		code     string
		expected Locale
		ok       bool
	}{
		{"en", LocaleEN, true},
		{"ru", LocaleRU, true},
		{"uk", LocaleUK, true},
		{"zh-hant", LocaleZH, true},
		{"pt-br", LocalePT, true},
		{"unknown", LocaleNone, false},
		{"", LocaleNone, false},
	}

	for _, tt := range tests {
		locale, ok := ParseLocale(tt.code)
		if ok != tt.ok || locale != tt.expected {
			t.Errorf("ParseLocale(%q) = (%v, %v), want (%v, %v)", tt.code, locale, ok, tt.expected, tt.ok)
		}
	}
}

func TestParseLocaleList(t *testing.T) {
	list := "en,ru,uk,unknown"
	locales := ParseLocaleList(list)
	expected := []Locale{LocaleEN, LocaleRU, LocaleUK}
	if len(locales) != len(expected) {
		t.Fatalf("ParseLocaleList(%q) = %v, want %v", list, locales, expected)
	}
	for i, l := range locales {
		if l != expected[i] {
			t.Errorf("ParseLocaleList: got %v at %d, want %v", l, i, expected[i])
		}
	}
}

func TestLocale_Code_Name_String(t *testing.T) {
	locale := LocaleRU
	if locale.Code() != "ru" {
		t.Errorf("LocaleRU.Code() = %q, want %q", locale.Code(), "ru")
	}
	if locale.Name() != "Russian" {
		t.Errorf("LocaleRU.Name() = %q, want %q", locale.Name(), "Russian")
	}
	if locale.String() != "ru" {
		t.Errorf("LocaleRU.String() = %q, want %q", locale.String(), "ru")
	}
}
