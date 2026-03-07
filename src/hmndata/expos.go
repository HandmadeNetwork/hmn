package hmndata

import (
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/utils"
)

type Expo struct {
	Event

	TemplateName string
}

var HMNExpo2026 = Expo{
	Event: Event{
		StartTime:   time.Date(2026, 6, 6, 9, 30, 0, 0, utils.Must1(time.LoadLocation("America/Vancouver"))),
		EndTime:     time.Date(2026, 6, 7, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Vancouver"))),
		Name:        "Handmade Network Expo 2026",
		Description: "A day of demos, brainstorming, and socializing in Vancouver, BC.",
		Slug:        "EXPO2026",
		UrlSlug:     "vancouver-2026",

		IndexUrl:         hmnurl.BuildExpo("vancouver-2026", ""),
		TicketSuccessUrl: hmnurl.BuildExpoTicketPurchaseSuccess("vancouver-2026"),
		TicketCancelUrl:  hmnurl.BuildExpo("vancouver-2026", "cancel"),
	},

	TemplateName: "2026",
}

var AllExpos = []Expo{
	HMNExpo2026,
}

var LatestExpo = HMNExpo2026
