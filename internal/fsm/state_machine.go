package fsm

import (
	"context"
	"sync"
)

type State int
type StateTask func(arg interface{}) []State

const (
	Start State = iota
	End
)

type StateMachine struct {
	state State
	tasks sync.Map
	sub   *StateMachine
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		state: Start,
		tasks: sync.Map{},
		sub:   nil,
	}
}

func (s *StateMachine) On(state State, task StateTask) {
	s.tasks.Store(state, task)
}

func (s *StateMachine) Run(context context.Context) {
	for s.state != End {
		select {
		case <-context.Done():
			return
		default:
			value, exist := s.tasks.Load(s.state)
			if !exist {
				s.state = End
				continue
			}

			task, ok := value.(StateTask)
			if !ok {
				s.state = End
				continue
			}

			states := task(1)
			if len(states) == 0 {
				s.state = End
				continue
			}

			if len(states) < 2 {
				s.state = states[0]
			} else {
				s.state = states[0]
				// for _, _ := range states {
				// }
			}
		}
	}
}
