package perf

import (
	"context"
	"time"
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
	for rp.EndBlock() {
	}
	rp.End = time.Now()
}

func (rp *RequestPerf) Checkpoint(category, description string) {
	now := time.Now()
	checkpoint := PerfBlock{
		Start:       now,
		End:         now,
		Category:    category,
		Description: description,
	}
	rp.Blocks = append(rp.Blocks, checkpoint)
}

func (rp *RequestPerf) StartBlock(category, description string) {
	now := time.Now()
	checkpoint := PerfBlock{
		Start:       now,
		End:         time.Time{},
		Category:    category,
		Description: description,
	}
	rp.Blocks = append(rp.Blocks, checkpoint)
}

func (rp *RequestPerf) EndBlock() bool {
	for i := len(rp.Blocks) - 1; i >= 0; i -= 1 {
		if rp.Blocks[i].End.Equal(time.Time{}) {
			rp.Blocks[i].End = time.Now()
			return true
		}
	}
	return false
}

func (rp *RequestPerf) MsFromStart(block *PerfBlock) float64 {
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
	Done        <-chan struct{}
	RequestCopy chan<- (chan<- PerfStorage)
}

func RunPerfCollector(ctx context.Context) *PerfCollector {
	in := make(chan RequestPerf)
	done := make(chan struct{})
	requestCopy := make(chan (chan<- PerfStorage))

	var storage PerfStorage
	// TODO(asaf): Load history from file

	go func() {
		defer close(done)

		for {
			select {
			case perf := <-in:
				storage.AllRequests = append(storage.AllRequests, perf)
				// TODO(asaf): Write to file
			case resultChan := <-requestCopy:
				resultChan <- storage
			case <-ctx.Done():
				return
			}
		}
	}()

	perfCollector := PerfCollector{
		In:          in,
		Done:        done,
		RequestCopy: requestCopy,
	}
	return &perfCollector
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
