package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInstaTicker(t *testing.T) {
	t.Run("normal behavior", func(t *testing.T) {
		it := NewInstaTicker(time.Millisecond * 200)
		var ticks []time.Time
		go func() {
			for tick := range it.C {
				t.Logf("Tick! %v", tick)
				ticks = append(ticks, tick)
			}
		}()
		time.Sleep(time.Millisecond * 500)
		assert.Len(t, ticks, 3)

		it.Stop()
		time.Sleep(time.Millisecond * 500)
		assert.Len(t, ticks, 3)

		select {
		case <-it.C:
			assert.Fail(t, "No more ticks should be received after stop")
		default:
		}
	})
	t.Run("stop", func(t *testing.T) {
		t.Run("never consumed a tick", func(t *testing.T) {
			it := NewInstaTicker(time.Second * 100)
			it.Stop()
		})
		t.Run("consumed initial tick", func(t *testing.T) {
			it := NewInstaTicker(time.Millisecond * 50)
			<-it.C
			it.Stop()
		})
		t.Run("consumed one ticker tick", func(t *testing.T) {
			it := NewInstaTicker(time.Millisecond * 50)
			<-it.C
			<-it.C
			it.Stop()
		})
		t.Run("consumed two ticker ticks", func(t *testing.T) {
			it := NewInstaTicker(time.Millisecond * 50)
			<-it.C
			<-it.C
			<-it.C
			it.Stop()
		})
	})
}
