package website

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

type TicketsEventMetadata struct {
	Slug                    string
	MaxTickets              int
	MaxReserved             int
	SoldTickets             int
	ReservedTickets         int
	AvailableForPurchase    int
	AvailableForReservation int
	PriceAmount             string
	PriceCurrency           string
}

func (metadata *TicketsEventMetadata) RemainingTicketsForSale() int {
	reserved := utils.Max(metadata.MaxReserved, metadata.ReservedTickets)
	remaining := metadata.MaxTickets - reserved - metadata.SoldTickets
	return remaining
}

type TicketTemplateData struct {
	UUID                  string
	User                  *templates.User
	OwnerName             string
	OwnerEmail            string
	PurchasePriceAmount   string
	PruchasePriceCurrency string
	Reserved              bool
	AllocationDate        time.Time
	Note                  string
}

type TicketsAdminEventTemplateData struct {
	Event    hmndata.Event
	Metadata TicketsEventMetadata
	Tickets  []TicketTemplateData
	Url      string
}

func TicketsAdmin(c *RequestContext) ResponseData {
	type TicketsTemplateData struct {
		templates.BaseData
		TicketEvents []TicketsAdminEventTemplateData
	}
	data := TicketsTemplateData{
		BaseData: getBaseData(c, "Admin ticket dashboard", nil),
	}
	for _, e := range hmndata.AllTicketEvents {
		metadata, err := fetchTicketMetadata(c, c.Conn, e.Slug)
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
		TicketsEvent TicketsAdminEventTemplateData
	}
	urlSlug := c.PathParams["urlslug"]

	event, found := findTicketEventBySlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	metadata, err := fetchTicketMetadata(c, c.Conn, event.Slug)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch ticket metadata"))
	}

	data := TicketsEventTemplateData{
		BaseData: getBaseData(c, fmt.Sprintf("Admin ticket dashboard - %s", event.Name), nil),
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
		return c.RejectRequest(fmt.Sprintf("Event with slug %s not found", eventUrlSlug))
	}

	maxTicketsStr := c.Req.Form.Get("max_tickets")
	maxReservedTicketsStr := c.Req.Form.Get("max_reserved_tickets")
	priceStr := c.Req.Form.Get("price_amount")

	maxTickets, err := strconv.Atoi(maxTicketsStr)
	if err != nil {
		return c.RejectRequest("Max tickets must be a number")
	}
	maxReservedTickets, err := strconv.Atoi(maxReservedTicketsStr)
	if err != nil {
		return c.RejectRequest("Max reserved tickets must be a number")
	}

	_, err = strconv.ParseFloat(priceStr, 32)
	if err != nil {
		return c.RejectRequest("Price must be a number")
	}

	_, err = c.Conn.Exec(c,
		`
		INSERT INTO ticket_metadata
		(slug, max_tickets, max_reserved, price_amount, price_currency)
		VALUES
		($1, $2, $3, $4, 'USD')
		ON CONFLICT (slug) DO UPDATE SET
			max_tickets = EXCLUDED.max_tickets,
			max_reserved = EXCLUDED.max_reserved,
			price_amount = EXCLUDED.price_amount
		`,
		event.Slug,
		maxTickets,
		maxReservedTickets,
		priceStr,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to update event ticket metadata"))
	}

	return c.Redirect(hmnurl.BuildTicketsAdminEvent(eventUrlSlug), http.StatusSeeOther)
}

func TicketsEventBuy(c *RequestContext) ResponseData {
	urlSlug := c.PathParams["urlslug"]
	event, found := findTicketEventBySlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	if time.Now().After(event.StartTime) {
		return c.RejectRequest("We're no longer selling tickets for this event.")
	}

	metadata, err := fetchTicketMetadata(c, c.Conn, event.Slug)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to fetch event ticket metadata"))
	}

	if metadata.RemainingTicketsForSale() <= 0 {
		return c.RejectRequest("We've run out of tickets for this event.")
	}

	// Create stripe checkout stuff and redirect to payment
	return FourOhFour(c)
}

func TicketsEventBuyPurchased(c *RequestContext) ResponseData {
	urlSlug := c.PathParams["urlslug"]
	event, found := findTicketEventBySlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	metadata, err := fetchTicketMetadata(c, c.Conn, event.Slug)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to fetch event ticket metadata"))
	}

	if metadata.RemainingTicketsForSale() <= 0 {
		return c.RejectRequest("We've run out of tickets for this event. You were not charged.")
	}
	// Check remaining tickets
	// Verify and capture payment
	// Add ticket and save payment data
	// Send email with details
	return FourOhFour(c)
}

func fetchTicketMetadata(ctx context.Context, conn db.ConnOrTx, slug string) (TicketsEventMetadata, error) {
	type TicketMetadata struct {
		Slug          string `db:"slug"`
		MaxTickets    int    `db:"max_tickets"`
		MaxReserved   int    `db:"max_reserved"`
		PriceAmount   string `db:"price_amount"`
		PriceCurrency string `db:"price_currency"`
	}
	metadata, err := db.QueryOne[TicketMetadata](ctx, conn,
		`
		SELECT $columns
		FROM ticket_metadata
		WHERE slug = $1
		`,
		slug,
	)

	if err != nil {
		if err != db.NotFound {
			return TicketsEventMetadata{}, oops.New(err, "Failed to fetch ticket metadata")
		}
	}

	result := TicketsEventMetadata{
		Slug:          slug,
		PriceCurrency: "USD",
	}

	if metadata != nil {
		result.Slug = slug
		result.MaxTickets = metadata.MaxTickets
		result.MaxReserved = metadata.MaxReserved
		result.PriceAmount = metadata.PriceAmount
		result.PriceCurrency = metadata.PriceCurrency

		type TicketAllocations struct {
			SoldTickets     int `db:"COUNT(*) FILTER (WHERE reserved = FALSE) AS sold_tickets"`
			ReservedTickets int `db:"COUNT(*) FILTER (WHERE reserved = TRUE) AS reserved_tickets"`
		}

		allocs, err := db.QueryOne[TicketAllocations](ctx, conn,
			`
			SELECT $columns
			FROM ticket
			WHERE event_slug = $1
			`,
			slug,
		)

		result.ReservedTickets = allocs.ReservedTickets
		result.SoldTickets = allocs.SoldTickets

		if err != nil {
			return TicketsEventMetadata{}, oops.New(err, "Failed to fetch ticket statistics")
		}
	}

	return result, nil
}

func findTicketEventBySlug(urlSlug string) (hmndata.Event, bool) {
	for _, e := range hmndata.AllTicketEvents {
		if e.UrlSlug == urlSlug {
			return e, true
		}
	}

	return hmndata.Event{}, false
}
