//go:build !windows

package main

import (
	"os"
	"syscall"
)

// terminationSignals lists the signals that should trigger a graceful shutdown.
// SIGTERM is used by most process managers (systemd, kubernetes) to request shutdown.
var terminationSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
