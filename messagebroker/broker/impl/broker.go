package impl

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/triasbrata/higo/messagebroker/broker"
)

type brokerConAMQP struct {
	name         string
	uri          string
	vhost        string
	locale       string
	chanMax      uint16
	heartBeat    time.Duration
	retryCounter atomic.Int64
}
type conConfig struct {
	amqp *brokerConAMQP
}

type brk struct {
	cfg          conConfig
	restartTimer time.Duration
}

type brokerConfig func(inst *brk)

func CreateNewBroker(config ...brokerConfig) (broker.Broker, error) {
	inst := &brk{}
	for _, cfx := range config {
		cfx(inst)
	}
	rvCon := reflect.ValueOf(inst.cfg)
	haveConfig := 0
	for fIndex := range rvCon.NumField() {
		if !rvCon.Field(fIndex).IsNil() {
			haveConfig++
		}
	}
	if haveConfig == 0 {
		return nil, fmt.Errorf("cant open connection when no configuration was passed")
	}
	return inst, nil
}
