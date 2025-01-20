package website

import (
	"encoding/json"
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func Perfmon(c *RequestContext) ResponseData {
	b := c.Perf.StartBlock("PERF", "Requesting perf data")
	perfData := c.PerfCollector.GetPerfCopy()
	b.End()

	type FlameItem struct {
		Offset      int64
		Duration    int64
		Category    string
		Description string
		Children    []*FlameItem
		End         time.Time  `json:"-"`
		Parent      *FlameItem `json:"-"`
	}

	type PerfRecord struct {
		Route     string
		Path      string
		Duration  int64
		Breakdown *FlameItem
	}

	type PerfmonData struct {
		templates.BaseData
		PerfRecordsJSON string
	}

	var perfJSON []byte
	{
		b := c.Perf.StartBlock("PERF", "Processing perf data")
		defer b.End()

		perfRecords := []PerfRecord{}
		for _, item := range perfData.AllRequests {
			record := PerfRecord{
				Route:    item.Route,
				Path:     item.Path,
				Duration: item.End.Sub(item.Start).Microseconds(),
				Breakdown: &FlameItem{
					Offset:   0,
					Duration: item.End.Sub(item.Start).Microseconds(),
					End:      item.End,
				},
			}

			parent := record.Breakdown
			for _, block := range item.Blocks {
				for parent.Parent != nil && block.End.After(parent.End) {
					parent = parent.Parent
				}
				flame := FlameItem{
					Offset:      block.Start.Sub(item.Start).Microseconds(),
					Duration:    block.End.Sub(block.Start).Microseconds(),
					Category:    block.Category,
					Description: block.Description,
					End:         block.End,
					Parent:      parent,
				}

				parent.Children = append(parent.Children, &flame)
				parent = &flame
			}

			perfRecords = append(perfRecords, record)
		}

		var err error
		perfJSON, err = json.Marshal(perfRecords)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to marshal json"))
		}

		b.End()
	}

	var res ResponseData
	res.MustWriteTemplate("perfmon.html", PerfmonData{
		BaseData:        getBaseDataAutocrumb(c, "Perfmon"),
		PerfRecordsJSON: string(perfJSON),
	}, c.Perf)
	return res
}
