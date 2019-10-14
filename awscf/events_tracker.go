package awscf

type EventsTrack struct {
	stack *Stack

	seenEvents      map[string]bool
	trackingStarted bool
}

func (et *EventsTrack) FreshEvents() (StackEvents, error) {
	emptyEvents := StackEvents{}

	// IMPORTANT: newer events appear at the beginning of a slice
	events, err := et.stack.Events()
	if err != nil {
		return emptyEvents, err
	}

	if !et.trackingStarted {
		et.seenEvents = make(map[string]bool, len(events))

		for _, e := range events {
			et.seenEvents[e.ID] = true
		}

		et.trackingStarted = true

		return emptyEvents, nil
	}

	if len(events) == 0 {
		return emptyEvents, nil
	}

	freshEvents := make(StackEvents, 0, len(events))

	for _, e := range events {
		if _, ok := et.seenEvents[e.ID]; !ok {
			freshEvents = append(freshEvents, e)
			et.seenEvents[e.ID] = true
		}
	}

	return freshEvents, nil
}
