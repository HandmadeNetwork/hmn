package hmndata

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
