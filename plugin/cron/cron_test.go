//nolint:all
package cron

import (
	"bytes"
	"log"
	"sync"
	"time"
)

// Test utilities for cron package (minimal, fast tests only)

// syncWriter is a thread-safe bytes.Buffer
type syncWriter struct {
	wr bytes.Buffer
	m  sync.Mutex
}

func (sw *syncWriter) Write(data []byte) (n int, err error) {
	sw.m.Lock()
	n, err = sw.wr.Write(data)
	sw.m.Unlock()
	return
}

func (sw *syncWriter) String() string {
	sw.m.Lock()
	defer sw.m.Unlock()
	return sw.wr.String()
}

func newBufLogger(sw *syncWriter) Logger {
	return PrintfLogger(log.New(sw, "", log.LstdFlags))
}

// OneSecond is used for quick timing tests
const OneSecond = 1100 * time.Millisecond // Slightly over 1s for @every 1s tasks
