package gotrans

// Translation represents a translated field value for a specific entity and locale.
// ID is typically auto-incremented by the database and may be omitted for insert operations.
// Entity is the type name (e.g., "product"), EntityID identifies the specific entity instance.
// Field maps to the translatable field name (e.g., "title", "description").
// Locale specifies the language variant.
// Value contains the translated text.
type Translation struct {
	// ID is the database primary key (omitted for inserts).
	ID int
	// Entity is the entity type name (e.g., "product", "parameter").
	Entity string
	// EntityID is the unique identifier of the entity instance.
	EntityID int
	// Field is the database field identifier (e.g., "title", "description").
	Field string
	// Locale is the language variant for this translation.
	Locale Locale
	// Value contains the translated text.
	Value string
}
