package perf

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestProfile(t *testing.T) {
	go func() {
		for {
			var buf [8192]byte
			rand.Read(buf[:])
		}
	}()

	rp := CreateRollingProfile(1 * 1024)
	rp.Profile(100 * time.Millisecond)
}
