package models

import "time"

type UserPayment struct {
	ID                int       `db:"id"`
	UserID            int       `db:"user_id"`
	StripeInvoiceID   *string   `db:"stripe_invoice_id"`
	AmountCents       int       `db:"amount_cents"`
	Currency          string    `db:"currency"`
	PaymentMethodType *string   `db:"payment_method_type"`
	CardLast4         *string   `db:"card_last4"`
	CardBrand         *string   `db:"card_brand"`
	PaidAt            time.Time `db:"paid_at"`
	StripeFeeCents    *int      `db:"stripe_fee_cents"`
	NetAmountCents    *int      `db:"net_amount_cents"`
}
