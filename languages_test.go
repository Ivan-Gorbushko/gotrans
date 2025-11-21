package gotrans

import (
	"testing"
)

func TestParseLang(t *testing.T) {
	tests := []struct {
		code     string
		expected Lang
		ok       bool
	}{
		{"en", LangEN, true},
		{"ru", LangRU, true},
		{"uk", LangUK, true},
		{"zh-hant", LangZH, true},
		{"pt-br", LangPT, true},
		{"unknown", LangNone, false},
		{"", LangNone, false},
	}

	for _, tt := range tests {
		lang, ok := ParseLang(tt.code)
		if ok != tt.ok || lang != tt.expected {
			t.Errorf("ParseLang(%q) = (%v, %v), want (%v, %v)", tt.code, lang, ok, tt.expected, tt.ok)
		}
	}
}

func TestParseLangList(t *testing.T) {
	list := "en,ru,uk,unknown"
	langs := ParseLangList(list)
	expected := []Lang{LangEN, LangRU, LangUK}
	if len(langs) != len(expected) {
		t.Fatalf("ParseLangList(%q) = %v, want %v", list, langs, expected)
	}
	for i, l := range langs {
		if l != expected[i] {
			t.Errorf("ParseLangList: got %v at %d, want %v", l, i, expected[i])
		}
	}
}

func TestLang_Code_Name_String(t *testing.T) {
	lang := LangRU
	if lang.Code() != "ru" {
		t.Errorf("LangRU.Code() = %q, want %q", lang.Code(), "ru")
	}
	if lang.Name() != "Russian" {
		t.Errorf("LangRU.Name() = %q, want %q", lang.Name(), "Russian")
	}
	if lang.String() != "ru" {
		t.Errorf("LangRU.String() = %q, want %q", lang.String(), "ru")
	}
}
