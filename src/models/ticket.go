package models

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID uuid.UUID `db:"id"`

	EventSlug   string `db:"event_slug"`
	OwnerUserID *int   `db:"user_id"`
	OwnerName   string `db:"name"`
	OwnerEmail  string `db:"email"`

	CheckedIn bool

	// Whether the ticket has been reserved ahead of time and therefore has no corresponding payment.
	Reserved bool `db:"reserved"`

	// Whether the ticket is pending payment. Tickets that are pending will be automatically deleted
	// when their PurchaseDate is too old to leave room for new ticket purchasers. When this happens,
	// the reference to the Stripe payment intent will be deleted, meaning that any new events that
	// come in for that payment intent can be safely disregarded.
	Pending bool `db:"pending"`

	// Generally the date when the ticket was purchased / reserved.
	PurchaseDate time.Time `db:"purchase_date"`

	StripeCheckoutSessionID string `db:"stripe_cs_id"`
	StripePaymentIntentID   string `db:"stripe_pi_id"`
	StripePriceAmount       int64  `db:"stripe_price_amount"`
	StripePriceCurrency     string `db:"stripe_price_currency"`

	Notes string `db:"notes"`

	// Not a field on the ticket table. Must be filled in by fetching functions.
	OwnerUsername string
}

type TicketMetadata struct {
	EventSlug   string `db:"slug"`
	MaxTickets  int    `db:"max_tickets"`
	MaxReserved int    `db:"max_reserved"`

	// If a presale is active, then the ticket purchase link will be active but Buy buttons will be
	// marked "coming soon".
	Presale bool `db:"presale"`

	StripePriceID       string `db:"stripe_price_id"`
	StripePriceAmount   int64  `db:"stripe_price_amount"`
	StripePriceCurrency string `db:"stripe_price_currency"`
}
