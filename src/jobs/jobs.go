package jobs

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/logging"
	"github.com/rs/zerolog"
)

/*
 * This package provides utilities for running and waiting on background tasks
 * in your application. It standardizes a few aspects of channels and contexts
 * to provide a system that makes it easy to run background jobs that can be
 * canceled and shut down gracefully.
 */

// A Job is used to handle and track the completion of an asynchronous or
// background task. See ExampleJob in example_test.go for a complete example of
// how to structure and use a Job.
type Job struct {
	Name   string
	Ctx    context.Context
	Logger zerolog.Logger
	cancel func()
	done   chan struct{}
}

func New(name string) *Job {
	logger := logging.With().Str("job", name).Logger()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logging.AttachLoggerToContext(&logger, ctx)
	return &Job{
		Name:   name,
		Ctx:    ctx,
		Logger: logger,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// Sends a cancel signal to the Job, indicating that it should finish its work
// and shut down. Internally, this cancels the Job's context. Expected to be
// called from outside the job, e.g. when shutting down the application.
func (j *Job) Cancel() {
	j.cancel()
}

// Returns a channel that can be waited on to receive a Cancel signal from
// outside (that is, when Cancel() has been called).
func (j *Job) Canceled() <-chan struct{} {
	return j.Ctx.Done()
}

// Marks the Job as finished, indicating that its work is completely done.
// Expected to be called internally by the job code when the work is complete.
func (j *Job) Finish() *Job {
	close(j.done)
	return j
}

// Returns a channel that can be waited on to tell when the Job is finished
// (that is, when Finish() has been called). Expected to be used outside the
// job to tell when work is complete.
func (j *Job) Finished() <-chan struct{} {
	return j.done
}

// A utility for running and canceling multiple jobs at once. Because this type
// is simply a slice of Jobs, you can construct it using normal slice syntax.
type Jobs []*Job

// Cancels all tracked jobs, giving them a chance to finish gracefully. Will
// return when all jobs finish or when the timeout expires, whichever comes
// first. Returns a list of all jobs that did not finish on time.
func (jobs Jobs) CancelAndWait(timeout time.Duration) []string {
	allDoneChan := make(chan struct{})
	for _, job := range jobs {
		job.Cancel()
	}
	timer := time.NewTimer(timeout)

	go func() {
		for _, job := range jobs {
			<-job.Finished()
		}
		close(allDoneChan)
	}()

	select {
	case <-timer.C:
		return jobs.ListUnfinished()
	case <-allDoneChan:
		return nil
	}
}

func (jobs Jobs) ListUnfinished() []string {
	unfinished := []string{}
	for _, job := range jobs {
		select {
		case <-job.Finished():
			continue
		default:
			unfinished = append(unfinished, job.Name)
		}
	}
	return unfinished
}
