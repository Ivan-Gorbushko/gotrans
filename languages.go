package gotrans

import "strings"

type Lang int16

const (
	LangNone Lang = iota
	LangSQ        // Albanian
	LangAR        // Arabic
	LangAZ        // Azerbaijani
	LangBS        // Bosnian
	LangBG        // Bulgarian
	LangZH        // Chinese
	LangHR        // Croatian
	LangCS        // Czech
	LangDA        // Danish
	LangNL        // Dutch
	LangEN        // English
	LangET        // Estonian
	LangFI        // Finnish
	LangFR        // French
	LangKA        // Georgian
	LangDE        // German
	LangEL        // Greek
	LangHE        // Hebrew
	LangHU        // Hungarian
	LangID        // Indonesian
	LangJA        // Japanese
	LangKK        // Kazakh
	LangKO        // Korean
	LangLV        // Latvian
	LangLT        // Lithuanian
	LangMK        // Macedonian
	LangNO        // Norwegian
	LangPL        // Polish
	LangPT        // Portuguese
	LangRO        // Romanian
	LangRU        // Russian
	LangSR        // Serbian
	LangSK        // Slovak
	LangSL        // Slovenian
	LangES        // Spanish
	LangSV        // Swedish
	LangTH        // Thai
	LangTR        // Turkish
	LangUK        // Ukrainian
	LangVI        // Vietnamese
	LangIT        // Italian
)

type langInfo struct {
	code string
	name string
}

// Main ISO-639-1 registry
var languages = map[Lang]langInfo{
	LangSQ: {"sq", "Albanian"},
	LangAR: {"ar", "Arabic"},
	LangAZ: {"az", "Azerbaijani"},
	LangBS: {"bs", "Bosnian"},
	LangBG: {"bg", "Bulgarian"},
	LangZH: {"zh", "Chinese"},
	LangHR: {"hr", "Croatian"},
	LangCS: {"cs", "Czech"},
	LangDA: {"da", "Danish"},
	LangNL: {"nl", "Dutch"},
	LangEN: {"en", "English"},
	LangET: {"et", "Estonian"},
	LangFI: {"fi", "Finnish"},
	LangFR: {"fr", "French"},
	LangKA: {"ka", "Georgian"},
	LangDE: {"de", "German"},
	LangEL: {"el", "Greek"},
	LangHE: {"he", "Hebrew"},
	LangHU: {"hu", "Hungarian"},
	LangID: {"id", "Indonesia"},
	LangJA: {"ja", "Japanese"},
	LangKK: {"kk", "Kazakh"},
	LangKO: {"ko", "Korean"},
	LangLV: {"lv", "Latvian"},
	LangLT: {"lt", "Lithuanian"},
	LangMK: {"mk", "Macedonian"},
	LangNO: {"no", "Norwegian"},
	LangPL: {"pl", "Polish"},
	LangPT: {"pt", "Portuguese"},
	LangRO: {"ro", "Romanian"},
	LangRU: {"ru", "Russian"},
	LangSR: {"sr", "Serbian"},
	LangSK: {"sk", "Slovak"},
	LangSL: {"sl", "Slovenian"},
	LangES: {"es", "Spanish"},
	LangSV: {"sv", "Swedish"},
	LangTH: {"th", "Thai"},
	LangTR: {"tr", "Turkish"},
	LangUK: {"uk", "Ukrainian"},
	LangVI: {"vi", "Vietnamese"},
	LangIT: {"it", "Italian"},
}

// Acceptable aliases, including BCP47 fallbacks
var aliases = map[string]Lang{
	"zh-hant": LangZH,
	"zh-hans": LangZH,
	"sr-latn": LangSR,
	"pt-br":   LangPT,
}

// Map lookup table
var codeToLang = func() map[string]Lang {
	m := make(map[string]Lang)
	for l, info := range languages {
		m[info.code] = l
	}
	for alias, lang := range aliases {
		m[alias] = lang
	}
	return m
}()

// ParseLang returns a Lang enum from a language code (ISO-639-1).
// Returns (LangNone, false) for unknown codes.
func ParseLang(code string) (Lang, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	l, ok := codeToLang[code]
	return l, ok
}

// ParseLangList converts "en,ru,uk" into []Lang.
func ParseLangList(list string) []Lang {
	parts := strings.Split(list, ",")
	res := make([]Lang, 0, len(parts))

	for _, p := range parts {
		if lang, ok := ParseLang(p); ok {
			res = append(res, lang)
		}
	}
	return res
}

// Code returns the ISO-639-1 code for a language.
func (l Lang) Code() string {
	if info, ok := languages[l]; ok {
		return info.code
	}
	return ""
}

// Name returns the human-readable language name.
func (l Lang) Name() string {
	if info, ok := languages[l]; ok {
		return info.name
	}
	return ""
}

func (l Lang) String() string {
	return l.Code()
}
