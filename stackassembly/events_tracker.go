package stackassembly

import (
	"time"
)

type EventsTracker struct {
	sleep time.Duration
}

// StartTracking starts event tracking
func (et *EventsTracker) StartTracking(stack *Stack) (chan StackEvent, func()) {
	eventsCh := make(chan StackEvent)
	stopCh := make(chan bool)

	latestEventID := ""
	events, _ := stack.Events()

	if len(events) > 0 {
		latestEventID = events[0].ID
	}

	sleep := et.sleep

	if sleep == 0 {
		sleep = 3 * time.Second
	}
	go func() {
		for {
			latestEventID = et.publishEvents(stack, eventsCh, latestEventID)

			select {
			case <-stopCh:
				et.publishEvents(stack, eventsCh, latestEventID)
				close(eventsCh)
				return
			default:
			}

			time.Sleep(sleep)
		}
	}()

	return eventsCh, func() { close(stopCh) }
}

func (et *EventsTracker) publishEvents(stack *Stack, eventsCh chan StackEvent, sinceEventID string) string {
	events, _ := stack.Events()

	lastEventIndex := 0

	for i, e := range events {
		if e.ID == sinceEventID {
			lastEventIndex = i
			break
		}
	}

	events = events[:lastEventIndex]

	if len(events) > 0 {
		sinceEventID = events[0].ID

		for _, e := range reverse(events) {
			eventsCh <- e
		}
	}

	return sinceEventID
}

func reverse(s []StackEvent) []StackEvent {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
