package types

import "strings"

type SimpleEventQuery struct {
	Name string
}

func (s *SimpleEventQuery) String() string {
	return s.Name
}

func (s *SimpleEventQuery) Matches(tags map[string]interface{}) bool {
	if len(tags) == 0 {
		return false
	}

	v, ok := tags[EventTypeKey]
	if !ok {
		return false
	}

	queryValueIndex := strings.Index(s.Name, "=")
	if queryValueIndex <= 0 {
		return false
	}
	queryValue := s.Name[queryValueIndex+1:]
	value, ok := v.(string)
	if !ok { // if value from tags is not string
		return false
	}
	return queryValue == value
}
