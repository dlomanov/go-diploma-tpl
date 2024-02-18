package entity

import "slices"

type (
	Event  string
	Events []Event
)

func NewEvents() Events {
	return make([]Event, 0)
}

func (es Events) Contains(e Event) bool {
	return slices.Contains(es, e)
}

func (es Events) Empty() bool {
	return len(es) == 0
}
