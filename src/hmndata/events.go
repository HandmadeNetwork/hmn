package hmndata

import (
	"time"

	"git.handmade.network/hmn/hmn/src/utils"
)

type Event struct {
	StartTime, EndTime             time.Time
	BannerStartTime, BannerEndTime time.Time

	Name        string
	Description string // NOTE(asaf): Used by opengraph
	Image       string // NOTE(ben): Used by opengraph
	Slug        string
	UrlSlug     string
	Url         string

	IndexUrl         string
	TicketSuccessUrl string
	TicketCancelUrl  string

	// Some events have specific template patterns that are disambiguated by this slug.
	TemplateName string
}

func (ev Event) Within(t time.Time) bool {
	return ev.WithinGrace(t, 0, 0)
}

func (ev Event) WithinBanner(t time.Time) bool {
	// NOTE(ben): This is kind of jank, but basically we assume that any event
	// without an explicit banner start and end time is a jam and we use these
	// common jam grace period variables for it, which is dumb but idk. We should
	// probably just fill in all the banner things on existing jams and delete
	// these constants.
	if ev.BannerStartTime.IsZero() {
		return ev.WithinGrace(t, JamBannerGraceBefore, JamBannerGraceAfter)
	}
	return ev.BannerStartTime.Before(t) && t.Before(ev.BannerEndTime)
}

func (ev Event) WithinGrace(t time.Time, before, after time.Duration) bool {
	return ev.StartTime.Add(-before).Before(t) && t.Before(ev.EndTime.Add(after))
}

type EventTimespans struct {
	DaysUntilStart int
	DaysUntilEnd   int

	Pre        bool
	During     bool
	Post       bool
	BeforeEnd  bool // Pre OR During
	AfterStart bool // During OR Post
}

func CalcTimespans(ev Event, t time.Time) EventTimespans {
	timespans := EventTimespans{
		DaysUntilStart: utils.DaysUntilT(ev.StartTime, t),
		DaysUntilEnd:   utils.DaysUntilT(ev.EndTime, t),
		Pre:            t.Before(ev.StartTime),
		During:         t.Before(ev.EndTime) && ev.StartTime.Before(t),
		Post:           ev.EndTime.Before(t),
	}

	timespans.BeforeEnd = timespans.Pre || timespans.During
	timespans.AfterStart = timespans.During || timespans.Post

	return timespans
}

// Old conferences
var HMS2022 = Event{
	StartTime: time.Date(2022, 11, 16, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	EndTime:   time.Date(2022, 11, 18, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
}

var HMS2023 = Event{
	StartTime: time.Date(2023, 11, 15, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	EndTime:   time.Date(2023, 11, 17, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
}

var HMBoston2023 = Event{
	StartTime: time.Date(2023, 8, 3, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	EndTime:   time.Date(2023, 8, 4, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
}

var HMS2024 = Event{
	StartTime: time.Date(2024, 11, 20, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	EndTime:   time.Date(2024, 11, 22, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
}

var HMBoston2024 = Event{
	StartTime: time.Date(2024, 8, 9, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	EndTime:   time.Date(2024, 8, 10, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
}
