package claws

import (
	"fmt"
	"time"

	"github.com/molecule-man/claws/cloudprov"
)

// EventsTracker tracks stack events
type EventsTracker struct {
	stackName     string
	cp            cloudprov.CloudProvider
	stopCh        chan bool
	latestEventID string
}

// StartTracking starts event tracking
func (et *EventsTracker) StartTracking() chan string {
	eventsCh := make(chan string)
	et.stopCh = make(chan bool)

	events := et.eventsSince("")

	et.latestEventID = events[0].ID

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
			time.Sleep(time.Second)
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

func (et *EventsTracker) publishEvents(eventsCh chan string) {
	events := et.eventsSince(et.latestEventID)

	if len(events) > 0 {
		et.latestEventID = events[0].ID

		for _, e := range reverse(events) {
			eventsCh <- fmt.Sprintf("[%s] [%s] [%s] %s",
				e.ResourceType, e.Status, e.LogicalResourceID, e.StatusReason,
			)
		}
	}
}

func reverse(s []cloudprov.StackEvent) []cloudprov.StackEvent {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
