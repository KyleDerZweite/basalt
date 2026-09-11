// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"sync"
	"testing"
)

func TestServiceCloseIsConcurrentAndIdempotent(t *testing.T) {
	service, err := NewService("test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	errors := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			ready.Done()
			<-start
			errors <- service.Close()
		}()
	}
	ready.Wait()
	close(start)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
}
