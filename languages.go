package gotrans

import "strings"

type Locale int16

const (
	LocaleNone Locale = iota
	LocaleSQ          // Albanian
	LocaleAR          // Arabic
	LocaleAZ          // Azerbaijani
	LocaleBS          // Bosnian
	LocaleBG          // Bulgarian
	LocaleZH          // Chinese
	LocaleHR          // Croatian
	LocaleCS          // Czech
	LocaleDA          // Danish
	LocaleNL          // Dutch
	LocaleEN          // English
	LocaleET          // Estonian
	LocaleFI          // Finnish
	LocaleFR          // French
	LocaleKA          // Georgian
	LocaleDE          // German
	LocaleEL          // Greek
	LocaleHE          // Hebrew
	LocaleHU          // Hungarian
	LocaleID          // Indonesian
	LocaleJA          // Japanese
	LocaleKK          // Kazakh
	LocaleKO          // Korean
	LocaleLV          // Latvian
	LocaleLT          // Lithuanian
	LocaleMK          // Macedonian
	LocaleNO          // Norwegian
	LocalePL          // Polish
	LocalePT          // Portuguese
	LocaleRO          // Romanian
	LocaleRU          // Russian
	LocaleSR          // Serbian
	LocaleSK          // Slovak
	LocaleSL          // Slovenian
	LocaleES          // Spanish
	LocaleSV          // Swedish
	LocaleTH          // Thai
	LocaleTR          // Turkish
	LocaleUK          // Ukrainian
	LocaleVI          // Vietnamese
	LocaleIT          // Italian
)

type langInfo struct {
	code string
	name string
}

// Main ISO-639-1 registry
var languages = map[Locale]langInfo{
	LocaleSQ: {"sq", "Albanian"},
	LocaleAR: {"ar", "Arabic"},
	LocaleAZ: {"az", "Azerbaijani"},
	LocaleBS: {"bs", "Bosnian"},
	LocaleBG: {"bg", "Bulgarian"},
	LocaleZH: {"zh", "Chinese"},
	LocaleHR: {"hr", "Croatian"},
	LocaleCS: {"cs", "Czech"},
	LocaleDA: {"da", "Danish"},
	LocaleNL: {"nl", "Dutch"},
	LocaleEN: {"en", "English"},
	LocaleET: {"et", "Estonian"},
	LocaleFI: {"fi", "Finnish"},
	LocaleFR: {"fr", "French"},
	LocaleKA: {"ka", "Georgian"},
	LocaleDE: {"de", "German"},
	LocaleEL: {"el", "Greek"},
	LocaleHE: {"he", "Hebrew"},
	LocaleHU: {"hu", "Hungarian"},
	LocaleID: {"id", "Indonesia"},
	LocaleJA: {"ja", "Japanese"},
	LocaleKK: {"kk", "Kazakh"},
	LocaleKO: {"ko", "Korean"},
	LocaleLV: {"lv", "Latvian"},
	LocaleLT: {"lt", "Lithuanian"},
	LocaleMK: {"mk", "Macedonian"},
	LocaleNO: {"no", "Norwegian"},
	LocalePL: {"pl", "Polish"},
	LocalePT: {"pt", "Portuguese"},
	LocaleRO: {"ro", "Romanian"},
	LocaleRU: {"ru", "Russian"},
	LocaleSR: {"sr", "Serbian"},
	LocaleSK: {"sk", "Slovak"},
	LocaleSL: {"sl", "Slovenian"},
	LocaleES: {"es", "Spanish"},
	LocaleSV: {"sv", "Swedish"},
	LocaleTH: {"th", "Thai"},
	LocaleTR: {"tr", "Turkish"},
	LocaleUK: {"uk", "Ukrainian"},
	LocaleVI: {"vi", "Vietnamese"},
	LocaleIT: {"it", "Italian"},
}

// Acceptable aliases, including BCP47 fallbacks
var aliases = map[string]Locale{
	"zh-hant": LocaleZH,
	"zh-hans": LocaleZH,
	"sr-latn": LocaleSR,
	"pt-br":   LocalePT,
}

// Map lookup table
var codeToLocale = func() map[string]Locale {
	m := make(map[string]Locale)
	for l, info := range languages {
		m[info.code] = l
	}
	for alias, locale := range aliases {
		m[alias] = locale
	}
	return m
}()

// ParseLocale returns a Locale enum from a language code (ISO-639-1).
// Returns (LocaleNone, false) for unknown codes.
func ParseLocale(code string) (Locale, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	l, ok := codeToLocale[code]
	return l, ok
}

// ParseLocaleList converts "en,ru,uk" into []Locale.
func ParseLocaleList(list string) []Locale {
	parts := strings.Split(list, ",")
	res := make([]Locale, 0, len(parts))

	for _, p := range parts {
		if locale, ok := ParseLocale(p); ok {
			res = append(res, locale)
		}
	}
	return res
}

// Code returns the ISO-639-1 code for a language.
func (l Locale) Code() string {
	if info, ok := languages[l]; ok {
		return info.code
	}
	return ""
}

// Name returns the human-readable language name.
func (l Locale) Name() string {
	if info, ok := languages[l]; ok {
		return info.name
	}
	return ""
}

func (l Locale) String() string {
	return l.Code()
}
