package orchestrator

import (
	"log/slog"
	"sync"
)

// EventDispatcher ensures thread-safe, sequential event delivery to the callback.
type EventDispatcher struct {
	callback EventCallback
	eventCh  chan event
	wg       sync.WaitGroup
	closed   bool
	mu       sync.Mutex
	traceID  string
}

type event struct {
	Type string
	Data string
}

// NewEventDispatcher creates a new event dispatcher.
func NewEventDispatcher(traceID string, callback EventCallback) *EventDispatcher {
	if callback == nil {
		return &EventDispatcher{callback: nil, traceID: traceID}
	}

	d := &EventDispatcher{
		callback: callback,
		eventCh:  make(chan event, 100),
		traceID:  traceID,
	}

	d.wg.Add(1)
	go d.dispatchLoop()

	return d
}

func (d *EventDispatcher) dispatchLoop() {
	defer d.wg.Done()
	for e := range d.eventCh {
		// Recover from panic in callback to protect dispatcher loop
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event dispatcher: recovered from panic", "panic", r, "trace_id", d.traceID)
				}
			}()
			d.callback(e.Type, e.Data)
		}()
	}
}

// Send sends an event strictly sequentially.
// Uses non-blocking send to prevent executor goroutine from stalling on slow consumers.
func (d *EventDispatcher) Send(eventType, eventData string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.callback == nil || d.closed {
		return
	}

	// Use non-blocking send with select to prevent backpressure
	// If channel is full, log warning and drop event (better than blocking executor)
	select {
	case d.eventCh <- event{Type: eventType, Data: eventData}:
		// Event sent successfully
	default:
		// Channel full - log warning and drop event to prevent executor stall
		slog.Warn("event dispatcher: channel full, dropping event",
			"trace_id", d.traceID,
			"event_type", eventType,
			"buffer_size", cap(d.eventCh),
		)
	}
}

// Close closes the dispatcher and waits for all events to be processed.
func (d *EventDispatcher) Close() {
	d.mu.Lock()
	if d.callback == nil || d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	d.mu.Unlock()

	close(d.eventCh)
	d.wg.Wait()
}
