package gotrans

// Translation represents a translated field value for a specific entity and locale.
type Translation struct {
	ID       int
	Entity   string
	EntityID int
	Field    string
	Locale   Locale
	Value    string
}
