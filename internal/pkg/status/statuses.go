package status

import (
	"errors"
	"strings"
)

type State string

const (
	StateAuthorized State = "Authorized"
	StateDeclined   State = "Declined"
)

var (
	ErrInvalidState = errors.New("invalid payment state")

	allStates = map[string]State{
		strings.ToLower(string(StateAuthorized)): StateAuthorized,
		strings.ToLower(string(StateDeclined)):   StateDeclined,
	}
)

func NewState(state string) (State, error) {
	state = strings.ToLower(state)
	_, ok := allStates[state]
	if !ok {
		return "", ErrInvalidState
	}
	return allStates[state], nil
}

func StateFromIsAuthorized(isAuthorized bool) State {
	if isAuthorized {
		return StateAuthorized
	}
	return StateDeclined
}

func (s State) String() string {
	return string(s)
}

func (s State) IsValid() bool {
	_, ok := allStates[strings.ToLower(string(s))]
	return ok
}
