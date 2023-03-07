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

type Event struct {
	StartTime, EndTime time.Time
}

type Jam struct {
	Event
	Name string
	Slug string
}

var WRJ2021 = Jam{
	Event: Event{
		StartTime: time.Date(2021, 9, 27, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2021, 10, 4, 0, 0, 0, 0, time.UTC),
	},
	Name: "Wheel Reinvention Jam 2021",
	Slug: "WRJ2021",
}

var WRJ2022 = Jam{
	Event: Event{
		StartTime: time.Date(2022, 8, 15, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
		EndTime:   time.Date(2022, 8, 22, 8, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	},
	Name: "Wheel Reinvention Jam 2022",
	Slug: "WRJ2022",
}

var VJ2023 = Jam{
	Event: Event{
		StartTime: time.Date(2023, 4, 14, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 4, 17, 0, 0, 0, 0, time.UTC),
	},
	Name: "Visibility Jam 2023",
	Slug: "VJ2023",
}

var HMS2022 = Event{
	StartTime: time.Date(2022, 11, 16, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
	EndTime:   time.Date(2022, 11, 18, 0, 0, 0, 0, utils.Must1(time.LoadLocation("America/Los_Angeles"))),
}

var AllJams = []Jam{WRJ2021, WRJ2022}

func CurrentJam() *Jam {
	now := time.Now()
	for i, jam := range AllJams {
		if jam.StartTime.Before(now) && now.Before(jam.EndTime) {
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
		SELECT $columns
		FROM jam_project
		WHERE project_id = $1
		`,
		projectId,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch jams for project")
	}

	currentJam := CurrentJam()
	foundCurrent := false
	for i, _ := range jamProjects {
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
