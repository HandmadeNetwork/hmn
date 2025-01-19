package calendar

import (
	"bytes"
	"context"
	"crypto/sha1"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/emersion/go-ical"
)

type RawCalendarData struct {
	Name string
	Url  string
	Data []byte
	Hash [sha1.Size]byte
}

type CalendarEvent struct {
	ID        string
	Name      string
	Desc      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	CalName   string
}

var unifiedCalendar *ical.Calendar
var rawCalendarData = make([]*RawCalendarData, 0)
var cachedICals = make(map[string][]byte)
var httpClient = http.Client{}

// NOTE(asaf): Passing an empty array for selectedCals returns all cals
func GetICal(selectedCals []string) ([]byte, error) {
	if unifiedCalendar == nil {
		return nil, oops.New(nil, "No calendar")
	}
	sort.Strings(selectedCals)
	cacheKey := strings.Join(selectedCals, "##")
	cachedICal, ok := cachedICals[cacheKey]
	if ok {
		return cachedICal, nil
	}

	var cal *ical.Calendar
	if len(selectedCals) == 0 {
		cal = unifiedCalendar
	} else {
		cal = newHMNCalendar()
		for _, child := range unifiedCalendar.Children {
			include := true

			if child.Name == ical.CompEvent {
				calName, _ := child.Props.Text(ical.PropComment)
				if calName != "" {
					found := false
					for _, s := range selectedCals {
						if calName == s {
							found = true
						}
					}
					if !found {
						include = false
					}
				}
			}

			if include {
				cal.Children = append(cal.Children, child)
			}
		}
	}
	var calBytes []byte
	if len(cal.Children) > 0 {
		var buffer bytes.Buffer
		err := ical.NewEncoder(&buffer).Encode(cal)
		if err != nil {
			return nil, oops.New(err, "Failed to encode calendar to iCal")
		}
		calBytes = buffer.Bytes()
	} else {
		calBytes = emptyCalendarString()
	}
	cachedICals[cacheKey] = calBytes
	return calBytes, nil
}

func GetFutureEvents() []CalendarEvent {
	if unifiedCalendar == nil {
		return nil
	}

	futureEvents := make([]CalendarEvent, 0)
	eventObjects := unifiedCalendar.Events()
	now := time.Now()
	lastTime := now.Add(time.Hour * 24 * 365)
	for _, ev := range eventObjects {
		summary, err := ev.Props.Text(ical.PropSummary)
		if err != nil {
			logging.Error().Err(err).Msg("Failed to get summary for calendar event")
			continue
		}

		startTime, err := ev.DateTimeStart(nil)
		if err != nil {
			logging.Error().Err(err).Str("Event name", summary).Msg("Failed to get start time for calendar event")
			continue
		}

		var evTimes []time.Time
		set, err := ev.RecurrenceSet(nil)
		if err != nil {
			logging.Error().Err(err).Str("Event name", summary).Msg("Failed to get recurrence set for calendar event")
			continue
		}
		if set != nil {
			evTimes = set.Between(now, lastTime, true)
		} else if startTime.After(now) {
			evTimes = []time.Time{startTime}
		}

		if len(evTimes) == 0 {
			continue
		}

		desc, err := ev.Props.Text(ical.PropDescription)
		if err != nil {
			logging.Error().Err(err).Str("Event name", summary).Msg("Failed to get description for calendar event")
			continue
		}

		calName, _ := ev.Props.Text(ical.PropComment)

		uid, err := ev.Props.Text(ical.PropUID)
		if err != nil {
			logging.Error().Err(err).Str("Event name", summary).Msg("Failed to get uid for calendar event")
			continue
		}

		endTime, err := ev.DateTimeStart(nil)
		if err != nil {
			logging.Error().Err(err).Str("Event name", summary).Msg("Failed to get end time for calendar event")
			continue
		}

		evDuration := endTime.Sub(startTime)

		for _, t := range evTimes {
			futureEvents = append(futureEvents, CalendarEvent{
				ID:        uid,
				Name:      summary,
				Desc:      desc,
				StartTime: t,
				EndTime:   t.Add(evDuration),
				Duration:  evDuration,
				CalName:   calName,
			})
		}
	}
	sort.Slice(futureEvents, func(i, j int) bool {
		return futureEvents[i].StartTime.Before(futureEvents[j].StartTime)
	})
	return futureEvents
}

func MonitorCalendars() *jobs.Job {
	job := jobs.New("calendar monitor")
	log := job.Logger

	if len(config.Config.Calendars) == 0 {
		log.Info().Msg("No calendars specified in config")
		return job.Finish()
	}

	go func() {
		defer func() {
			log.Info().Msg("Shutting down calendar monitor")
			job.Finish()
		}()
		log.Info().Msg("Running calendar monitor")

		monitorTimer := time.NewTimer(time.Second)

		for {
			select {
			case <-monitorTimer.C:
				err := func() (err error) {
					defer utils.RecoverPanicAsError(&err)

					ReloadCalendars(job.Ctx)

					return nil
				}()
				if err != nil {
					logging.Error().Err(err).Msg("Panicked in MonitorCalendars")
				}
				monitorTimer.Reset(60 * time.Minute)
			case <-job.Canceled():
				return
			}
		}
	}()

	return job
}

func ReloadCalendars(ctx context.Context) {
	log := logging.ExtractLogger(ctx)

	// Download calendars
	calChan := make(chan RawCalendarData, len(config.Config.Calendars))
	var wg sync.WaitGroup
	wg.Add(len(config.Config.Calendars))
	for _, c := range config.Config.Calendars {
		go func(cal config.CalendarSource) {
			defer func() {
				wg.Done()
				logging.LogPanics(log)
			}()
			calUrl := cal.Url
			req, err := http.NewRequestWithContext(ctx, "GET", calUrl, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create request for calendar fetch")
				return
			}
			res, err := httpClient.Do(req)
			if err != nil {
				log.Error().Err(err).Str("Url", calUrl).Msg("Failed to fetch calendar")
				return
			}
			if res.StatusCode > 299 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/calendar") {
				log.Error().Str("Url", calUrl).Str("Status", res.Status).Msg("Failed to fetch calendar")
				io.ReadAll(res.Body)
				res.Body.Close()
				return
			}

			data, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				log.Error().Err(err).Str("Url", calUrl).Msg("Failed to fetch calendar")
				return
			}

			calChan <- RawCalendarData{Name: cal.Name, Url: calUrl, Data: data}
		}(c)
	}
	wg.Wait()
	newRawCalendarData := make([]*RawCalendarData, 0, len(config.Config.Calendars))
Collect:
	for {
		select {
		case d := <-calChan:
			newRawCalendarData = append(newRawCalendarData, &d)
		default:
			break Collect
		}
	}

	// Diff calendars
	same := true
	for _, n := range newRawCalendarData {
		n.Hash = sha1.Sum(n.Data)
	}

	sort.Slice(newRawCalendarData, func(i, j int) bool {
		return newRawCalendarData[i].Name < newRawCalendarData[j].Name
	})

	if len(newRawCalendarData) != len(rawCalendarData) {
		same = false
	} else {
		for i := range newRawCalendarData {
			newData := newRawCalendarData[i]
			oldData := rawCalendarData[i]
			if newData.Name != oldData.Name {
				same = false
				break
			}
			if newData.Hash != oldData.Hash {
				same = false
				break
			}
		}
	}

	if same {
		return
	}

	// Unify calendars and clear cache
	rawCalendarData = newRawCalendarData
	cachedICals = make(map[string][]byte)

	unified := newHMNCalendar()

	var timezones []string

	for _, calData := range rawCalendarData {
		decoder := ical.NewDecoder(bytes.NewReader(calData.Data))
		calNameProp := ical.NewProp(ical.PropComment)
		calNameProp.SetText(calData.Name)
		for {
			cal, err := decoder.Decode()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Error().Err(err).Str("Url", calData.Url).Msg("Failed to parse calendar")
				break
			}

			for _, child := range cal.Children {
				if child.Name == ical.CompTimezone {
					tzid, err := child.Props.Text(ical.PropTimezoneID)
					if err != nil {
						found := false
						for _, s := range timezones {
							if s == tzid {
								found = true
							}
						}
						if found {
							continue
						} else {
							timezones = append(timezones, tzid)
						}
					} else {
						continue
					}
				}
				if child.Name == ical.CompEvent {
					child.Props.Set(calNameProp)
				}
				unified.Children = append(unified.Children, child)
			}
		}
	}

	unifiedCalendar = unified
}

func newHMNCalendar() *ical.Calendar {
	cal := ical.NewCalendar()

	prodID := ical.NewProp(ical.PropProductID)
	prodID.SetText("Handmade Network")
	cal.Props.Set(prodID)

	version := ical.NewProp(ical.PropVersion)
	version.SetText("1.0")
	cal.Props.Set(version)

	name := ical.NewProp("X-WR-CALNAME")
	name.SetText("Handmade Network")
	cal.Props.Set(name)

	return cal
}

// NOTE(asaf): The ical library we're using doesn't like encoding empty calendars, so we have to do this manually.
func emptyCalendarString() []byte {
	empty := `BEGIN:VCALENDAR
VERSION:1.0
PRODID:Handmade Network
X-WR-CALNAME:Handmade Network
END:VCALENDAR
	`

	return []byte(empty)
}
