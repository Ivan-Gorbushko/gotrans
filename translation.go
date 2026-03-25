package gotrans

// Translation represents a translated field value for a specific entity and locale
type Translation struct {
	ID       int
	Entity   string
	EntityID int
	Field    string
	locale   Locale
	Value    string
}

// GetLocale returns the locale of the translation
func (t Translation) GetLocale() Locale {
	return t.locale
}

// NewTranslation creates a new Translation instance with the given parameters
func NewTranslation(id int, entity string, entityID int, field string, locale Locale, value string) Translation {
	return Translation{
		ID:       id,
		Entity:   entity,
		EntityID: entityID,
		Field:    field,
		locale:   locale,
		Value:    value,
	}
}

