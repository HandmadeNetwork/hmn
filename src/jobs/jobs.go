package jobs

type Job struct {
	C    <-chan struct{}
	rawC chan struct{}
}

func New() Job {
	return newFromChannel(make(chan struct{}))
}

func (j *Job) Done() {
	close(j.rawC)
}

// Combines multiple jobs into one.
func Zip(jobs ...Job) Job {
	out := make(chan struct{})
	go func() {
		for _, job := range jobs {
			<-job.C
		}
		close(out)
	}()
	return newFromChannel(out)
}

// Returns a job that is already done.
func Noop() Job {
	job := New()
	job.Done()
	return job
}

func newFromChannel(c chan struct{}) Job {
	return Job{
		C:    c,
		rawC: c,
	}
}
