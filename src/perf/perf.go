package perf

import (
	"context"
	"fmt"
	"time"

	"git.handmade.network/hmn/hmn/src/jobs"
	"github.com/rs/zerolog"
)

type RequestPerf struct {
	Route  string
	Path   string // the path actually matched
	Method string
	Start  time.Time
	End    time.Time
	Blocks []PerfBlock
}

func MakeNewRequestPerf(route string, method string, path string) *RequestPerf {
	return &RequestPerf{
		Start:  time.Now(),
		Route:  route,
		Path:   path,
		Method: method,
	}
}

func (rp *RequestPerf) EndRequest() {
	if rp == nil {
		return
	}

	// Close any open blocks (arguably a bug! do better, user!)
	for i := len(rp.Blocks) - 1; i >= 0; i -= 1 {
		if rp.Blocks[i].End.Equal(time.Time{}) {
			rp.Blocks[i].End = time.Now()
		}
	}

	rp.End = time.Now()
}

func (rp *RequestPerf) Checkpoint(category, description string) {
	if rp == nil {
		return
	}

	now := time.Now()
	checkpoint := PerfBlock{
		Start:       now,
		End:         now,
		Category:    category,
		Description: description,
	}
	rp.Blocks = append(rp.Blocks, checkpoint)
}

type BlockHandle struct {
	rp    *RequestPerf
	ended bool
}

func (b *BlockHandle) End() {
	if b.rp == nil {
		return
	}
	if b.ended {
		return
	}

	b.ended = true
	for i := len(b.rp.Blocks) - 1; i >= 0; i -= 1 {
		if b.rp.Blocks[i].End.Equal(time.Time{}) {
			b.rp.Blocks[i].End = time.Now()
			return
		}
	}
}

func StartBlock(ctx context.Context, category, description string) *BlockHandle {
	p := ExtractPerf(ctx)
	return p.StartBlock(category, description)
}

func (rp *RequestPerf) StartBlock(category, description string) *BlockHandle {
	if rp == nil {
		return &BlockHandle{}
	}

	now := time.Now()
	checkpoint := PerfBlock{
		Start:       now,
		End:         time.Time{},
		Category:    category,
		Description: description,
	}
	rp.Blocks = append(rp.Blocks, checkpoint)

	return &BlockHandle{
		rp: rp,
	}
}

func (rp *RequestPerf) MsFromStart(block *PerfBlock) float64 {
	if rp == nil {
		return 0
	}

	return float64(block.Start.Sub(rp.Start).Nanoseconds()) / 1000 / 1000
}

type PerfBlock struct {
	Start       time.Time
	End         time.Time
	Category    string
	Description string
}

func (pb *PerfBlock) Duration() time.Duration {
	return pb.End.Sub(pb.Start)
}

func (pb *PerfBlock) DurationMs() float64 {
	return float64(pb.Duration().Nanoseconds()) / 1000 / 1000
}

type PerfStorage struct {
	AllRequests []RequestPerf
}

type PerfCollector struct {
	In          chan<- RequestPerf
	RequestCopy chan<- (chan<- PerfStorage)
}

func RunPerfCollector() (*PerfCollector, *jobs.Job) {
	in := make(chan RequestPerf, 100)
	job := jobs.New("perf collector")
	requestCopy := make(chan (chan<- PerfStorage))

	cleanupTicker := time.NewTicker(10 * time.Second)

	var storage PerfStorage
	// TODO(asaf): Load history from file

	go func() {
		for {
			select {
			case perf := <-in:
				storage.AllRequests = append(storage.AllRequests, perf)
				// TODO(asaf): Write to file
			case resultChan := <-requestCopy:
				storageCopy := make([]RequestPerf, len(storage.AllRequests))
				copy(storageCopy, storage.AllRequests)
				resultChan <- PerfStorage{
					storageCopy,
				}
			case <-cleanupTicker.C:
				if len(storage.AllRequests) > 1000 {
					storage.AllRequests = storage.AllRequests[len(storage.AllRequests)-800:]
				}
			case <-job.Canceled():
				job.Finish()
				return
			}
		}
	}()

	perfCollector := PerfCollector{
		In:          in,
		RequestCopy: requestCopy,
	}
	return &perfCollector, job
}

func (perfCollector *PerfCollector) SubmitRun(run *RequestPerf) {
	perfCollector.In <- *run
}

func (perfCollector *PerfCollector) GetPerfCopy() *PerfStorage {
	resultChan := make(chan PerfStorage)
	perfCollector.RequestCopy <- resultChan
	perfStorageCopy := <-resultChan
	return &perfStorageCopy
}

func LogPerf(perf *RequestPerf, log *zerolog.Event) {
	blockStack := make([]time.Time, 0)
	for i, block := range perf.Blocks {
		for len(blockStack) > 0 && block.End.After(blockStack[len(blockStack)-1]) {
			blockStack = blockStack[:len(blockStack)-1]
		}
		log.Str(fmt.Sprintf("[%4.d] At %9.2fms", i, perf.MsFromStart(&block)), fmt.Sprintf("%*.s[%s] %s (%.4fms)", len(blockStack)*2, "", block.Category, block.Description, block.DurationMs()))
		blockStack = append(blockStack, block.End)
	}
	log.Msg(fmt.Sprintf("Served [%s] %s in %.4fms", perf.Method, perf.Path, float64(perf.End.Sub(perf.Start).Nanoseconds())/1000/1000))
}

const PerfContextKey = "HMNPerf"

func ExtractPerf(ctx context.Context) *RequestPerf {
	iperf := ctx.Value(PerfContextKey)
	if iperf == nil {
		// NOTE(asaf): Returning a dummy perf so we don't crash if it's missing
		return MakeNewRequestPerf("PERF MISSING", "PERF MISSING", "PERF MISSING")
	}
	return iperf.(*RequestPerf)
}
