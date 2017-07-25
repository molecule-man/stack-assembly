package claws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

// EventsTracker tracks stack events
type EventsTracker struct {
	stackName     string
	cf            *cloudformation.CloudFormation
	stopCh        chan bool
	latestEventID *string
}

// StartTracking starts event tracking
func (et *EventsTracker) StartTracking() chan string {
	eventsCh := make(chan string)
	et.stopCh = make(chan bool)

	events := et.eventsSince(nil)

	et.latestEventID = events[0].EventId

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

func (et *EventsTracker) eventsSince(sinceEventID *string) []*cloudformation.StackEvent {
	events, err := et.cf.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(et.stackName),
	})

	if err != nil {
		return []*cloudformation.StackEvent{}
	}

	if sinceEventID == nil {
		return events.StackEvents
	}

	lastEventIndex := 0

	for i, e := range events.StackEvents {
		if *e.EventId == *sinceEventID {
			lastEventIndex = i
			break
		}
	}

	return events.StackEvents[:lastEventIndex]
}

func (et *EventsTracker) publishEvents(eventsCh chan string) {
	events := et.eventsSince(et.latestEventID)

	if len(events) > 0 {
		et.latestEventID = events[0].EventId

		for _, e := range reverse(events) {
			eventsCh <- fmt.Sprintf("[%s] [%s] [%s] %s",
				fromAwsString(e.ResourceType),
				fromAwsString(e.ResourceStatus),
				fromAwsString(e.LogicalResourceId),
				fromAwsString(e.ResourceStatusReason),
			)
		}
	}
}

func fromAwsString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func reverse(s []*cloudformation.StackEvent) []*cloudformation.StackEvent {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
