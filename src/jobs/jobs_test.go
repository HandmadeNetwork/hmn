package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrackerCancelAndWait(t *testing.T) {
	t.Run("finishes fast enough", func(t *testing.T) {
		testJobs := Jobs{
			FakeJob("Job A", time.Millisecond*100),
			FakeJob("Job B", time.Millisecond*200),
		}

		before := time.Now()
		unfinished := testJobs.CancelAndWait(time.Second * 1)
		after := time.Now()
		assert.WithinDuration(t, after, before, time.Millisecond*500, "tracker.Finish did not finish fast enough")
		assert.Len(t, unfinished, 0)
	})
	t.Run("reports unfinished jobs", func(t *testing.T) {
		testJobs := Jobs{
			FakeJob("Job A", time.Millisecond*100),
			FakeJob("Job B", time.Second*10),
		}

		unfinished := testJobs.CancelAndWait(time.Second * 1)
		assert.Equal(t, []string{"Job B"}, unfinished)
	})
}

func FakeJob(name string, timeout time.Duration) *Job {
	job := New(name)
	go func() {
		<-job.Ctx.Done()
		timer := time.NewTimer(timeout)
		<-timer.C
		job.Finish()
	}()
	return job
}
