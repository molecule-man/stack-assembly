package claws

import (
	"time"

	"github.com/molecule-man/claws/cloudprov"
)

// EventsTracker tracks stack events
type EventsTracker struct {
	stackName     string
	cp            cloudprov.CloudProvider
	stopCh        chan bool
	latestEventID string
	sleep         time.Duration
}

// StartTracking starts event tracking
func (et *EventsTracker) StartTracking() chan cloudprov.StackEvent {
	eventsCh := make(chan cloudprov.StackEvent)
	et.stopCh = make(chan bool)

	// @TODO using empty string here is ugly
	et.latestEventID = ""
	events := et.eventsSince(et.latestEventID)

	if len(events) > 0 {
		et.latestEventID = events[0].ID
	}

	go func() {
		for {
			et.publishEvents(eventsCh)

			select {
			case <-et.stopCh:
				et.publishEvents(eventsCh)
				close(eventsCh)
				return
			default:
			}
			time.Sleep(et.sleep)
		}
	}()

	return eventsCh
}

// StopTracking stops event tracking
func (et *EventsTracker) StopTracking() {
	close(et.stopCh)
}

func (et *EventsTracker) eventsSince(sinceEventID string) []cloudprov.StackEvent {
	events, err := et.cp.StackEvents(et.stackName)

	if err != nil {
		return []cloudprov.StackEvent{}
	}

	// @TODO come up with better name than sinceEventID
	if sinceEventID == "" {
		return events
	}

	lastEventIndex := 0

	for i, e := range events {
		if e.ID == sinceEventID {
			lastEventIndex = i
			break
		}
	}

	return events[:lastEventIndex]
}

func (et *EventsTracker) publishEvents(eventsCh chan cloudprov.StackEvent) {
	events := et.eventsSince(et.latestEventID)

	if len(events) > 0 {
		et.latestEventID = events[0].ID

		for _, e := range reverse(events) {
			eventsCh <- e
		}
	}
}

func reverse(s []cloudprov.StackEvent) []cloudprov.StackEvent {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
