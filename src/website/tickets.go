package website

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/price"
)

const TicketPendingExpiration = time.Minute * 31 // Stripe sets the minimum at 30 and I am paranoid about errors

type TicketPageCommon struct {
	TicketDescriptor string
}

var ticketPageTitles = [...]string{"Dominator", "Punisher", "Crusher", "Smasher", "Pulverizer", "Overlord", "Apocalypse"}

func getCommonTicketPageData() TicketPageCommon {
	return TicketPageCommon{
		TicketDescriptor: ticketPageTitles[rand.Intn(len(ticketPageTitles))],
	}
}

type TicketTemplateData struct {
	UUID                  string
	User                  *templates.User
	OwnerName             string
	OwnerEmail            string
	PurchasePriceAmount   string
	PurchasePriceCurrency string
	Reserved              bool
	AllocationDate        time.Time
	Note                  string
}

type TicketsAdminEventTemplateData struct {
	Event    hmndata.Event
	Metadata TicketMetadataForEvent
	Tickets  []TicketTemplateData
	Url      string
}

func TicketsAdmin(c *RequestContext) ResponseData {
	type TicketsTemplateData struct {
		templates.BaseData
		TicketPageCommon
		TicketEvents []TicketsAdminEventTemplateData
	}
	data := TicketsTemplateData{
		BaseData:         getBaseData(c, "Admin ticket dashboard", nil),
		TicketPageCommon: getCommonTicketPageData(),
	}
	for _, e := range hmndata.AllTicketEvents {
		metadata, err := fetchTicketMetadataForEvent(c, c.Conn, &e)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch ticket metadata"))
		}
		data.TicketEvents = append(data.TicketEvents, TicketsAdminEventTemplateData{
			Event:    e,
			Metadata: metadata,
			Url:      hmnurl.BuildTicketsAdminEvent(e.UrlSlug),
		})
	}

	var res ResponseData
	res.MustWriteTemplate("tickets_admin.html", data, c.Perf)
	return res
}

func TicketsAdminEvent(c *RequestContext) ResponseData {
	type TicketsEventTemplateData struct {
		templates.BaseData
		TicketPageCommon
		TicketsEvent TicketsAdminEventTemplateData
	}
	urlSlug := c.PathParams["urlslug"]

	event, found := findTicketEventBySlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	metadata, err := fetchTicketMetadataForEvent(c, c.Conn, &event)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch ticket metadata"))
	}

	data := TicketsEventTemplateData{
		BaseData:         getBaseData(c, fmt.Sprintf("Admin ticket dashboard - %s", event.Name), nil),
		TicketPageCommon: getCommonTicketPageData(),
		TicketsEvent: TicketsAdminEventTemplateData{
			Event:    event,
			Metadata: metadata,
			Url:      hmnurl.BuildTicketsAdminEvent(event.UrlSlug),
		},
	}

	var res ResponseData
	res.MustWriteTemplate("tickets_admin_event.html", data, c.Perf)
	return res
}

func TicketsAdminEventSubmit(c *RequestContext) ResponseData {
	err := c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to parse tickets admin form"))
	}

	eventUrlSlug := c.PathParams["urlslug"]
	event, found := findTicketEventBySlug(eventUrlSlug)
	if !found {
		return FourOhFour(c)
	}

	maxTicketsStr := c.Req.Form.Get("max_tickets")
	maxReservedTicketsStr := c.Req.Form.Get("max_reserved_tickets")
	priceID := c.Req.Form.Get("price_id")

	price, err := price.Get(priceID, &stripe.PriceParams{})
	if err != nil {
		c.Logger.Error().Err(err).Msg("Failed to retrieve Stripe price")
		return c.RejectRequest("Could not load price info from Stripe")
	}

	maxTickets, err := strconv.Atoi(maxTicketsStr)
	if err != nil {
		return c.RejectRequest("Max tickets must be a number")
	}
	maxReservedTickets, err := strconv.Atoi(maxReservedTicketsStr)
	if err != nil {
		return c.RejectRequest("Max reserved tickets must be a number")
	}

	_, err = c.Conn.Exec(c,
		`
		INSERT INTO ticket_metadata
		(slug, max_tickets, max_reserved, stripe_price_id, stripe_price_amount, stripe_price_currency)
		VALUES
		($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slug) DO UPDATE SET
			max_tickets = EXCLUDED.max_tickets,
			max_reserved = EXCLUDED.max_reserved,
			stripe_price_id = EXCLUDED.stripe_price_id,
			stripe_price_amount = EXCLUDED.stripe_price_amount,
			stripe_price_currency = EXCLUDED.stripe_price_currency
		`,
		event.Slug, maxTickets, maxReservedTickets, price.ID, price.UnitAmount, price.Currency,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to update event ticket metadata"))
	}

	return c.Redirect(hmnurl.BuildTicketsAdminEvent(eventUrlSlug), http.StatusSeeOther)
}

// A generic handler that takes an event slug and initiates a Stripe transaction for it. Will
// redirect the user to Stripe, which will then redirect the user back to the configured URL, which
// is expected to be event-specific.
//
// Because tickets are a limited-quantity thing, and we want to avoid issuing refunds, we first
// create a pending ticket in the DB (atomically!), then create a Stripe checkout session, which in
// turn creates a Stripe PaymentIntent (which tracks a specific payment attempt). We save the ID of
// the PaymentIntent to the pending ticket before sending the user off to the checkout form; that
// way, if wacky stuff happens with the payment, we can associate it precisely with the specific
// ticket attempt even if events come in late.
func TicketsPurchase(c *RequestContext) ResponseData {
	urlSlug := c.PathParams["urlslug"]
	event, found := findTicketEventBySlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	if time.Now().After(event.StartTime) {
		return c.RejectRequest("We're no longer selling tickets for this event.")
	}

	var pendingTicketID *uuid.UUID
	var metadata TicketMetadataForEvent
	tx, err := c.Conn.Begin(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start transaction"))
	}
	defer tx.Rollback(c)
	{
		// Check if the user already has a ticket
		existingTicket, err := fetchTicketForUser(c, tx, &event, c.CurrentUser.ID)
		if err == db.NotFound {
			// Good, they do not yet have a ticket
		} else if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check for existing ticket"))
		} else {
			if existingTicket.Pending {
				// This is a weird case where they must have started a new ticket-purchase flow without
				// completing a previous one. In this case it just makes the most sense to delete their
				// previous pending ticket and start over.
				_, err := c.Conn.Exec(c, `DELETE FROM ticket WHERE id = $1`, existingTicket.ID)
				if err != nil {
					return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete old pending ticket to make room for a new one"))
				}
				// We can now keep falling through the rest of the logic as if there were never a pending
				// ticket in the first place.
			} else {
				return c.RejectRequest("You already have a ticket for this event.")
			}
		}

		// Check if there are any tickets remaining
		metadata, err = fetchTicketMetadataForEvent(c, tx, &event)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch event ticket metadata"))
		}
		if metadata.RemainingTicketsForSale() <= 0 {
			return c.RejectRequest("We've run out of tickets for this event.")
		}

		// ...and check if we've even configured the event correctly.
		if metadata.StripePriceID == "" {
			return c.RejectRequest("The event has not been configured with Stripe yet. This is the admins' fault.")
		}

		// Create a pending ticket for this user (with no checkout session ID yet; we will fill that in
		// later after the transaction succeeds).
		pendingTicketID, err = db.QueryOne[uuid.UUID](c, tx,
			`
			INSERT INTO ticket (id, event_slug, pending, user_id, name, email)
			VALUES ($1, $2, TRUE, $3, $4, $5)
			RETURNING id
			`,
			uuid.New(), event.Slug, c.CurrentUser.ID, c.CurrentUser.BestName(), c.CurrentUser.Email,
		)
	}
	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create pending ticket"))
	}
	utils.Assert(pendingTicketID)

	// Defer a delete of the pending ticket in case things go wrong.
	deletePendingTicket := true
	defer func() {
		if deletePendingTicket {
			_, err := c.Conn.Exec(c, `DELETE FROM ticket WHERE id = $1`, *pendingTicketID)
			if err != nil {
				c.Logger.Error().Err(err).Msg("Failed to clean up bad pending ticket")
			}
		}
	}()

	// Create a Stripe checkout session
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(metadata.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},

		// We use the ticket ID as the client reference ID. However, the expected logic is to look up
		// the pending ticket using the CheckoutSession ID, then assert that the ticket ID matches the
		// client reference ID for sanity.
		ClientReferenceID: stripe.String(pendingTicketID.String()),
		CustomerEmail:     stripe.String(c.CurrentUser.Email),

		// We don't allow bank payments for ticket purchases due to the long transaction time.
		ExcludedPaymentMethodTypes: []*string{stripe.String("us_bank_account")},

		// We use Stripe's checkout session expirations to drive the cancellation of pending tickets.
		// On checkout session expiration, if the associated ticket is pending, we delete it. This
		// saves us from running yet another background job.
		ExpiresAt: stripe.Int64(time.Now().Add(TicketPendingExpiration).Unix()),

		SuccessURL: stripe.String(event.TicketSuccessUrl),
		CancelURL:  stripe.String(event.TicketCancelUrl),
	}
	result, err := session.New(params)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create Stripe checkout session"))
	}

	// Save the checkout session's PaymentIntent ID to the pending ticket before we send the user off
	// to Stripe.
	_, err = c.Conn.Exec(c,
		`
		UPDATE ticket SET stripe_cs_id = $2
		WHERE id = $1
		`,
		*pendingTicketID, result.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save checkout session ID on pending ticket"))
	}

	// Finally the user can go pay.
	deletePendingTicket = false
	return c.Redirect(result.URL, http.StatusSeeOther)
}

func confirmStripeTicketPurchase(ctx context.Context, conn db.ConnOrTx, session *stripe.CheckoutSession, ticket *models.Ticket) error {
	event, ok := findTicketEventBySlug(ticket.EventSlug)
	if !ok {
		return oops.New(nil, "no event found for paid ticket!! (id: %s, slug: %s)", ticket.ID, ticket.EventSlug)
	}

	// Sanity checks!
	{
		if ticket.ID.String() != session.ClientReferenceID {
			return oops.New(nil, "SANITY CHECK FAILED: ticket ID and client reference ID mismatch ('%s' vs. '%s')", ticket.ID.String(), session.ClientReferenceID)
		}
		if ticket.StripeCheckoutSessionID != session.ID {
			return oops.New(nil, "SANITY CHECK FAILED: ticket references other checkout session ('%s' vs. '%s')", ticket.StripeCheckoutSessionID, session.ID)
		}
	}

	_, err := conn.Exec(ctx,
		`
		UPDATE ticket SET pending = FALSE, stripe_pi_id = $2, stripe_price_amount = $3, stripe_price_currency = $4
		WHERE id = $1
		`,
		ticket.ID, session.PaymentIntent.ID, fmt.Sprintf("%d", session.AmountTotal), session.Currency,
	)
	if err != nil {
		return oops.New(err, "failed to update ticket after payment")
	}

	// TODO(ben): Send email with official information / QR code
	_ = event

	return nil
}

func cancelPendingTicketsForCheckoutSession(ctx context.Context, conn db.ConnOrTx, session *stripe.CheckoutSession) (int64, error) {
	logger := logging.ExtractLogger(ctx).With().Str("sessionID", session.ID).Logger()

	foo, err := conn.Exec(ctx,
		`
		DELETE FROM ticket
		WHERE stripe_cs_id = $1 AND pending = TRUE
		`,
		session.ID,
	)
	if err != nil {
		return 0, oops.New(err, "failed to delete tickets for checkout session")
	}

	if foo.RowsAffected() > 1 {
		logger.Warn().Int64("RowsAffected", foo.RowsAffected()).Msg("had multiple tickets for a single checkout session; this should not be possible")
	}

	return foo.RowsAffected(), nil
}

func findTicketEventBySlug(slugOrUrlSlug string) (hmndata.Event, bool) {
	for _, e := range hmndata.AllTicketEvents {
		if e.Slug == slugOrUrlSlug || e.UrlSlug == slugOrUrlSlug {
			return e, true
		}
	}

	return hmndata.Event{}, false
}

type TicketMetadataForEvent struct {
	models.TicketMetadata

	SoldTickets     int
	ReservedTickets int
}

func (metadata *TicketMetadataForEvent) RemainingTicketsForSale() int {
	reserved := utils.Max(metadata.MaxReserved, metadata.ReservedTickets)
	remaining := metadata.MaxTickets - reserved - metadata.SoldTickets
	return remaining
}

func fetchTicketMetadataForEvent(ctx context.Context, conn db.ConnOrTx, event *hmndata.Event) (TicketMetadataForEvent, error) {
	metadata, err := db.QueryOne[models.TicketMetadata](ctx, conn,
		`
		SELECT $columns
		FROM ticket_metadata
		WHERE slug = $1
		`,
		event.Slug,
	)
	if err == db.NotFound {
		// Return a default event, suitable for editing
		return TicketMetadataForEvent{
			TicketMetadata: models.TicketMetadata{
				EventSlug: event.Slug,
			},
		}, nil
	} else if err != nil {
		return TicketMetadataForEvent{}, oops.New(err, "failed to fetch ticket metadata")
	}
	utils.Assert(metadata)

	type ticketAllocations struct {
		SoldTickets     int `db:"COUNT(*) FILTER (WHERE reserved = FALSE) AS sold_tickets"`
		ReservedTickets int `db:"COUNT(*) FILTER (WHERE reserved = TRUE) AS reserved_tickets"`
	}
	allocs, err := db.QueryOne[ticketAllocations](ctx, conn,
		`
		SELECT $columns
		FROM ticket
		WHERE event_slug = $1
		`,
		event.Slug,
	)
	if err != nil {
		return TicketMetadataForEvent{}, oops.New(err, "failed to fetch ticket allocations")
	}

	return TicketMetadataForEvent{
		TicketMetadata:  *metadata,
		SoldTickets:     allocs.SoldTickets,
		ReservedTickets: allocs.ReservedTickets,
	}, nil
}

func fetchTicket(ctx context.Context, conn db.ConnOrTx, id string) (*models.Ticket, error) {
	ticket, err := db.QueryOne[models.Ticket](ctx, conn,
		`SELECT $columns FROM ticket WHERE id::TEXT = $1`,
		id,
	)
	if err == db.NotFound {
		return nil, err
	} else if err != nil {
		return nil, oops.New(err, "failed to look up ticket")
	}

	return ticket, nil
}

func fetchTicketForUser(ctx context.Context, conn db.ConnOrTx, event *hmndata.Event, userID int) (*models.Ticket, error) {
	ticket, err := db.QueryOne[models.Ticket](ctx, conn,
		`
		SELECT $columns
		FROM ticket
		WHERE event_slug = $1 AND user_id = $2
		`,
		event.Slug, userID,
	)
	if err == db.NotFound {
		return nil, err
	} else if err != nil {
		return nil, oops.New(err, "failed to look up user ticket")
	}

	return ticket, nil
}

func fetchTicketByCheckoutSessionID(ctx context.Context, conn db.ConnOrTx, id string) (*models.Ticket, error) {
	ticket, err := db.QueryOne[models.Ticket](ctx, conn,
		`
		SELECT $columns
		FROM ticket
		WHERE stripe_cs_id = $1
		`,
		id,
	)
	if err == db.NotFound {
		return nil, err
	} else if err != nil {
		return nil, oops.New(err, "failed to look up ticket by stripe ID")
	}

	return ticket, nil
}

func TicketSingle(c *RequestContext) ResponseData {
	ticket, err := fetchTicket(c, c.Conn, c.PathParams["id"])
	if err == db.NotFound {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up ticket for scanning"))
	}

	event, ok := findTicketEventBySlug(ticket.EventSlug)
	if !ok {
		c.Logger.Error().
			Str("ticketID", ticket.ID.String()).
			Str("eventSlug", ticket.EventSlug).
			Msg("Ticket event slug was invalid!")
		return c.RejectRequest("Your ticket is not associated with a valid event. This is our fault. Sorry.")
	}

	type TemplateData struct {
		templates.BaseData
		TicketID         string
		TicketCodeURL    string
		EventName        string
		EventDescription string
		EventURL         string
	}
	tmpl := TemplateData{
		BaseData:         getBaseData(c, fmt.Sprintf("%s Ticket", event.Name), nil),
		TicketID:         ticket.ID.String(),
		TicketCodeURL:    hmnurl.BuildTicketQRCode(ticket.ID.String()),
		EventName:        event.Name,
		EventDescription: event.Description,
		EventURL:         event.IndexUrl,
	}

	var res ResponseData
	res.MustWriteTemplate("tickets_single.html", tmpl, c.Perf)
	return res
}

func TicketQRCode(c *RequestContext) ResponseData {
	// NOTE(ben): We don't even bother to do a db lookup here. If someone provides a bad ticket ID,
	// our scanner will just reject it.
	codePNG := utils.Must1(qrcode.Encode(hmnurl.BuildTicketScanned(c.PathParams["id"]), qrcode.Medium, 1024))

	var res ResponseData
	res.Header().Add("Content-Type", "image/png")
	res.Write(codePNG)
	return res
}

func TicketScanned(c *RequestContext) ResponseData {
	if !c.CurrentUser.IsStaff {
		c.Redirect("https://www.youtube.com/watch?v=dQw4w9WgXcQ", http.StatusSeeOther)
	}

	// TODO(ben): Actually build ticket-scanning logic closer to the time of the event.
	return ResponseData{}
}
