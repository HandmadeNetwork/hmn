package perf

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"iter"
	"reflect"
	"runtime/pprof"
	"time"

	"git.handmade.network/hmn/hmn/src/perf/perfproto"
	"git.handmade.network/hmn/hmn/src/utils"
	"google.golang.org/protobuf/proto"
)

func (rp *RollingProfile) Profile(frequency time.Duration) {
	start := time.Now()
	var buf bytes.Buffer

	for range 10 {
		// Capture a new pprof CPU profile (gzipped protobuf data!!!)
		buf.Reset()
		pprof.StartCPUProfile(&buf)
		time.Sleep(frequency)
		pprof.StopCPUProfile()
		start = time.Now()

		// Parse the protobuf data
		var profile perfproto.Profile
		r := utils.Must1(gzip.NewReader(&buf))
		raw := utils.Must1(io.ReadAll(r))
		utils.Must(r.Close())
		utils.Must(proto.Unmarshal(raw, &profile))
		locs := make(map[uint64]*perfproto.Location, len(profile.Location))
		for _, loc := range profile.Location {
			locs[loc.Id] = loc
		}
		funcs := make(map[uint64]*perfproto.Function, len(profile.Function))
		for _, f := range profile.Function {
			funcs[f.Id] = f
		}

		// Gather all the new entries for the ringbuffer
		entries := make(map[Line]*RBEntry)
		for _, sample := range profile.Sample {
			// For CPU profiles, sample count is always value 0
			sampleCount := sample.Value[0]
			for _, locID := range sample.LocationId {
				loc := locs[locID]
				for _, protoLine := range loc.Line {
					f := funcs[protoLine.FunctionId]
					line := Line{
						Funcname: profile.StringTable[f.Name],
						Filename: profile.StringTable[f.Filename],
						Line:     protoLine.Line,
						Col:      protoLine.Column,
					}

					// Count up all of this profile's sample counts per location, making a
					// a new entry for each one.
					entry, ok := entries[line]
					if !ok {
						entry = &RBEntry{
							T:    start.UnixMicro(),
							Line: line,
						}
						entries[line] = entry
					}
					entry.SampleCount += sampleCount
				}
			}
		}

		// Add all entries to the ringbuffer
		for _, newEntry := range entries {
			rbEntry := rp.Insert()
			*rbEntry = *newEntry
		}
	}

	// TEMPORARY: Log all entries
	for entry := range rp.Iter() {
		t := time.UnixMicro(entry.T)
		fmt.Printf("%v (%d) %s %s:%d:%d\n", t, entry.SampleCount, entry.Line.Funcname, entry.Line.Filename, entry.Line.Line, entry.Line.Col)
	}
}

type RollingProfile struct {
	buf   []RBEntry
	cur   int
	count int
}

func (rp *RollingProfile) Insert() *RBEntry {
	res := &rp.buf[rp.cur]
	rp.cur = (rp.cur + 1) % len(rp.buf)
	if rp.count < len(rp.buf) {
		rp.count++
	}
	return res
}

type RBEntry struct {
	T           int64
	Line        Line
	SampleCount int64
}

type Line struct {
	Funcname  string
	Filename  string
	Line, Col int64
}

func CreateRollingProfile(sizeBytes int) RollingProfile {
	size := int(reflect.TypeOf(RBEntry{}).Size())
	cap := sizeBytes / size
	buf := make([]RBEntry, cap)
	return RollingProfile{
		buf:   buf,
		cur:   0,
		count: 0,
	}
}

func (rp *RollingProfile) Iter() iter.Seq[RBEntry] {
	return func(yield func(RBEntry) bool) {
		start := rp.cur % len(rp.buf)
		if rp.count < len(rp.buf) {
			start = 0
		}
		for i := 0; i < rp.count; i++ {
			if !yield(rp.buf[(start+i)%len(rp.buf)]) {
				break
			}
		}
	}
}
