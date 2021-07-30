package models

type ThreadType int

const (
	ThreadTypeProjectArticle ThreadType = iota + 1
	ThreadTypeForumPost
	_ // formerly occupied by static pages, RIP
	_ // formerly occupied by who the hell knows what, RIP
	_ // formerly occupied by the wiki, RIP
	_ // formerly occupied by library discussions, RIP
	ThreadTypePersonalArticle
)

type Thread struct {
	ID int `db:"id"`

	Type                  ThreadType `db:"type"`
	ProjectID             int        `db:"project_id"`
	SubforumID            *int       `db:"subforum_id"`
	PersonalArticleUserID *int       `db:"personal_article_user_id"`

	Title   string `db:"title"`
	Sticky  bool   `db:"sticky"`
	Locked  bool   `db:"locked"`
	Deleted bool   `db:"deleted"`

	FirstID int `db:"first_id"`
	LastID  int `db:"last_id"`
}
