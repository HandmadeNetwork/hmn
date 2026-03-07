package models

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID uuid.UUID `db:"id"`

	EventSlug   string `db:"event_slug"`
	OwnerUserID int    `db:"user_id"`
	OwnerName   string `db:"name"`
	OwnerEmail  string `db:"email"`

	// Whether the ticket has been reserved ahead of time and therefore has no corresponding payment.
	Reserved bool `db:"reserved"`

	// Whether the ticket is pending payment. Tickets that are pending will be automatically deleted
	// when their PurchaseDate is too old to leave room for new ticket purchasers. When this happens,
	// the reference to the Stripe payment intent will be deleted, meaning that any new events that
	// come in for that payment intent can be safely disregarded.
	Pending bool `db:"pending"`

	// Generally the date when the ticket was purchased / reserved.
	PurchaseDate time.Time `db:"purchase_date"`

	CheckoutSessionID string `db:"stripe_cs_id"`
	PriceAmount       string `db:"price_amount"`
	PriceCurrency     string `db:"price_currency"`

	Notes string `db:"notes"`
}

type TicketMetadata struct {
	EventSlug     string `db:"slug"`
	MaxTickets    int    `db:"max_tickets"`
	MaxReserved   int    `db:"max_reserved"`
	PriceAmount   string `db:"price_amount"`
	PriceCurrency string `db:"price_currency"`
}
