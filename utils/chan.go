package utils

import (
	"sync"
)

type StopChan struct {
	Chan chan struct{}
	sync.Once
}

func NewStopChan() *StopChan {
	return &StopChan{Chan: make(chan struct{})}
}

func (s *StopChan) CloseOnce() {
	s.Do(func() {
		close(s.Chan)
	})
}
