//go:build windows

package main

import (
	"os"
)

// terminationSignals lists the signals that should trigger a graceful shutdown.
// Windows primarily uses os.Interrupt (Ctrl+C).
var terminationSignals = []os.Signal{os.Interrupt}
