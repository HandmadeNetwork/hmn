package models

const HMNProjectID = 1

type Project struct {
	ID int `db:"id"`

	Slug        string `db:"slug"`
	Name        string `db:"name"`
	Blurb       string `db:"blurb"`
	Description string `db:"description"`

	Color1 string `db:"color_1"`
	Color2 string `db:"color_2"`
}

func (p *Project) IsHMN() bool {
	return p.ID == HMNProjectID
}
