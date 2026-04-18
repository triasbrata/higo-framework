package impl

import (
	"sync"

	"github.com/triasbrata/higo/messagebroker/manager"
	"github.com/triasbrata/higo/messagebroker/manager/connections"
)

type manCon[T manager.ShouldConnectionHave] struct {
	mutex     *sync.RWMutex
	con       T
	readyChan chan struct{}
}

// GetCon implements manager.Manager.
func (m *manCon[T]) GetCon() T {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.con
}

// SetCon implements manager.Manager.
func (m *manCon[T]) SetCon(con T) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.con = con
	go func() {
		m.readyChan <- struct{}{}
	}()
}

// Ready implements manager.Manager.
func (m *manCon[T]) Ready() <-chan struct{} {
	return m.readyChan
}

// SetConAMQP implements manager.Manager.
func (m *manCon[T]) Release() error {
	return m.con.Close()
}

func NewManager() manager.Manager[connections.ConnectionAMQP] {
	return &manCon[connections.ConnectionAMQP]{
		mutex:     &sync.RWMutex{},
		readyChan: make(chan struct{}, 1),
	}
}
