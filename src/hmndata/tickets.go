package hmndata

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
)

var AllTicketEvents = []Event{
	HMNExpo2026.Event,
}

func FindTicketEventBySlug(slugOrUrlSlug string) (Event, bool) {
	for _, e := range AllTicketEvents {
		if e.Slug == slugOrUrlSlug || e.UrlSlug == slugOrUrlSlug {
			return e, true
		}
	}

	return Event{}, false
}

func FetchTickets(ctx context.Context, conn db.ConnOrTx, event *Event) ([]models.Ticket, error) {
	type row struct {
		Ticket   models.Ticket `db:"ticket"`
		Username string        `db:"username"`
	}
	rows, err := db.Query[row](ctx, conn,
		`
		SELECT $columns
		FROM
			ticket
			JOIN hmn_user AS u ON ticket.user_id = u.id
		WHERE event_slug = $1
		ORDER BY purchase_date
		`,
		event.Slug,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch event tickets")
	}

	res := make([]models.Ticket, len(rows))
	for i := range rows {
		res[i] = rows[i].Ticket
		res[i].OwnerUsername = rows[i].Username
	}
	return res, nil
}

type TicketQuery struct {
	ID string

	EventSlug string
	UserID    int

	StripeCheckoutSessionID string
}

func FetchTicket(ctx context.Context, conn db.ConnOrTx, q TicketQuery) (*models.Ticket, error) {
	type row struct {
		Ticket   models.Ticket `db:"ticket"`
		Username string        `db:"username"`
	}

	var qb db.QueryBuilder
	qb.Add(`
		SELECT $columns
		FROM
			ticket
			JOIN hmn_user AS u ON ticket.user_id = u.id
		WHERE TRUE
	`)
	if q.ID != "" {
		qb.Add(`AND ticket.id::TEXT = $?`, q.ID)
	}
	if q.UserID != 0 {
		utils.Assert(q.EventSlug, "event slug is required when querying tickets by user ID")
		qb.Add(`AND event_slug = $? AND user_id = $?`, q.EventSlug, q.UserID)
	}
	if q.StripeCheckoutSessionID != "" {
		qb.Add(`AND stripe_cs_id = $?`, q.StripeCheckoutSessionID)
	}

	res, err := db.QueryOne[row](ctx, conn, qb.String(), qb.Args()...)
	if err == db.NotFound {
		return nil, err
	} else if err != nil {
		return nil, oops.New(err, "failed to fetch ticket")
	}

	t := &res.Ticket
	t.OwnerUsername = res.Username
	return t, nil
}
