package hmndata

import (
	"context"
	"sort"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
)

const JamProjectCreateGracePeriod = 7 * 24 * time.Hour

type Jam struct {
	Event
	Name        string
	Description string // NOTE(asaf): Used by opengraph
	Slug        string
	UrlSlug     string

	TemplateName string
	ForceDark    bool

	RecapStreamEmbedUrl string // NOTE(asaf): Youtube video, not twitch
}

var WRJ2021 = Jam{
	Event: Event{
		StartTime: time.Date(2021, 9, 27, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2021, 10, 4, 0, 0, 0, 0, time.UTC),
	},
	Name:    "Wheel Reinvention Jam 2021",
	Slug:    "WRJ2021",
	UrlSlug: "2021",
}

var WRJ2022 = Jam{
	Event: Event{
		StartTime: time.Date(2022, 8, 15, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
		EndTime:   time.Date(2022, 8, 22, 8, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	},
	Name:    "Wheel Reinvention Jam 2022",
	Slug:    "WRJ2022",
	UrlSlug: "2022",
}

var VJ2023 = Jam{
	Event: Event{
		StartTime: time.Date(2023, 4, 14, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 4, 17, 0, 0, 0, 0, time.UTC),
	},
	Name:    "Visibility Jam 2023",
	Slug:    "VJ2023",
	UrlSlug: "visibility-2023",
}

var WRJ2023 = Jam{
	Event: Event{
		StartTime: time.Date(2023, 9, 25, 10, 0, 0, 0, utils.Must1(time.LoadLocation("Europe/London"))),
		EndTime:   time.Date(2023, 10, 1, 20, 0, 0, 0, utils.Must1(time.LoadLocation("Europe/London"))),
	},
	Name:    "Wheel Reinvention Jam 2023",
	Slug:    "WRJ2023",
	UrlSlug: "2023",
}

var LJ2024 = Jam{
	Event: Event{
		StartTime: time.Date(2024, 3, 15, 17, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 3, 25, 0, 0, 0, 0, time.UTC),
	},
	Name:         "Learning Jam 2024",
	Description:  "A two-weekend jam where you dive deep into a topic, then share it with the rest of the community.",
	Slug:         "LJ2024",
	UrlSlug:      "learning-2024",
	TemplateName: "2024_lj",
	ForceDark:    true,
}

var VJ2024 = Jam{
	// Trying looser times this year.
	// Start: 6am Seattle / 8am Minneapolis / 1pm UTC / 2pm London / 4pm Jerusalem
	// End: 10pm Seattle / 12am Minneapolis / 5am UTC / 6am London / 8am Jerusalem
	Event: Event{
		StartTime: time.Date(2024, 7, 19, 13, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 7, 22, 5, 0, 0, 0, time.UTC),
	},
	Name:         "Visibility Jam 2024",
	Description:  "See things in a new way. July 19 - 21.",
	Slug:         "VJ2024",
	UrlSlug:      "visibility-2024",
	TemplateName: "2024_vj",
}

var WRJ2024 = Jam{
	Event: Event{
		StartTime: time.Date(2024, 9, 23, 13, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 9, 30, 5, 0, 0, 0, time.UTC),
	},
	Name:         "Wheel Reinvention Jam 2024",
	Description:  "A one-week jam where we build software from scratch. September 23 - 29 on the Handmade Network.",
	Slug:         "WRJ2024",
	UrlSlug:      "wheel-reinvention-2024",
	TemplateName: "2024_wrj",
}

var XRay2025 = Jam{
	Event: Event{
		StartTime: time.Date(2025, 6, 9, 13, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 16, 5, 0, 0, 0, time.UTC),
	},
	Name:         "X-Ray Jam 2025",
	Description:  "A jam to find out how software works on the inside. June 9 - 15 on the Handmade Network.",
	Slug:         "XRay2025",
	UrlSlug:      "x-ray-2025",
	TemplateName: "2025_xray",
	ForceDark:    true,
}

var WRJ2025 = Jam{
	Event: Event{
		StartTime: time.Date(2025, 9, 22, 13, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 9, 29, 5, 0, 0, 0, time.UTC),
	},
	Name:         "Wheel Reinvention Jam 2025",
	Description:  "A one-week jam to build software from scratch. September 22 - 28 on the Handmade Network.",
	Slug:         "WRJ2025",
	UrlSlug:      "wheel-reinvention-2025",
	TemplateName: "2025_wrj",
}

var AllJams = []Jam{
	WRJ2021,
	WRJ2022,
	VJ2023,
	WRJ2023,
	LJ2024,
	VJ2024,
	WRJ2024,
	XRay2025,
	WRJ2025,
}

var LatestJam = WRJ2025 // NOTE(asaf): The /jam route will redirect here

func CurrentJam() *Jam {
	now := time.Now()
	for i, jam := range AllJams {
		if jam.Event.Within(now) {
			return &AllJams[i]
		}
	}
	return nil
}

func UpcomingJam(window time.Duration) *Jam {
	now := time.Now()
	for i, jam := range AllJams {
		if jam.Event.WithinGrace(now, window, 0) {
			return &AllJams[i]
		}
	}
	return nil
}

func RecentJam(window time.Duration) *Jam {
	now := time.Now()
	for i, jam := range AllJams {
		if jam.Event.WithinGrace(now, 0, window) {
			return &AllJams[i]
		}
	}
	return nil
}

func JamBySlug(slug string) Jam {
	for _, jam := range AllJams {
		if jam.Slug == slug {
			return jam
		}
	}
	return Jam{Slug: slug}
}

func FetchJamsForProject(ctx context.Context, dbConn db.ConnOrTx, user *models.User, projectId int) ([]*models.JamProject, error) {
	jamProjects, err := db.Query[models.JamProject](ctx, dbConn,
		`
		---- Fetch jams for project
		SELECT $columns
		FROM jam_project
		WHERE project_id = $1
		`,
		projectId,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch jams for project")
	}

	currentJam := UpcomingJam(JamProjectCreateGracePeriod)
	foundCurrent := false
	for i := range jamProjects {
		jam := JamBySlug(jamProjects[i].JamSlug)
		jamProjects[i].JamName = jam.Name
		jamProjects[i].JamStartTime = jam.StartTime

		if currentJam != nil && currentJam.Slug == jamProjects[i].JamSlug {
			foundCurrent = true
		}
	}
	if currentJam != nil && !foundCurrent {
		jamProjects = append(jamProjects, &models.JamProject{
			ProjectID:     projectId,
			JamSlug:       currentJam.Slug,
			Participating: false,
			JamName:       currentJam.Name,
			JamStartTime:  currentJam.StartTime,
		})
	}

	if user != nil && user.IsStaff {
		for _, jam := range AllJams {
			found := false
			for _, jp := range jamProjects {
				if jp.JamSlug == jam.Slug {
					found = true
					break
				}
			}
			if !found {
				jamProjects = append(jamProjects, &models.JamProject{
					ProjectID:     projectId,
					JamSlug:       jam.Slug,
					Participating: false,
					JamName:       jam.Name,
					JamStartTime:  jam.StartTime,
				})
			}
		}
	}

	sort.Slice(jamProjects, func(i, j int) bool {
		return jamProjects[i].JamStartTime.Before(jamProjects[j].JamStartTime)
	})

	return jamProjects, nil
}
