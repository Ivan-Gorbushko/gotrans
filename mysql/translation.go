package mysql

type Translation struct {
	ID       int    `db:"id"`
	Entity   string `db:"entity"`
	EntityID int    `db:"entity_id"`
	Field    string `db:"field"`
	Locale   string `db:"locale"`
	Value    string `db:"value"`
}
