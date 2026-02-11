package gotrans

type Translation struct {
	ID       int
	Entity   string
	EntityID int
	Field    string
	Locale   Locale
	Value    string
}
