package models

type PersistentVar struct {
	Name  string `db:"name"`
	Value string `db:"value"`
}
