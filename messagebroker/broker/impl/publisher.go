package impl

import (
	"context"
	"fmt"
	"sync"

	"github.com/triasbrata/higo/messagebroker/broker"
	"github.com/triasbrata/higo/messagebroker/manager"
	"github.com/triasbrata/higo/messagebroker/manager/connections"
	"github.com/triasbrata/higo/messagebroker/publisher/amqp"
	"golang.org/x/sync/errgroup"

	"github.com/triasbrata/higo/messagebroker/publisher"
)

// Publisher implements broker.Broker.
func (b *brk) Publisher(ctx context.Context, builder broker.PubBuilder) (publisher.Publisher, error) {
	config := builder()
	switch {
	case config.Amqp:
		if b.cfg.amqp == nil {
			return nil, fmt.Errorf("configuration amqp was not found")
		}
		conHolders := make([]manager.Manager[connections.ConnectionAMQP], 0, 10)
		eg := errgroup.Group{}
		mut := sync.Mutex{}
		for cid := range 10 {
			eg.Go(func() error {
				conHolder, err := b.openConnectionAmqp(ctx, "-pub-", cid)
				if err != nil {
					return err
				}
				mut.Lock()
				conHolders = append(conHolders, conHolder)
				mut.Unlock()
				return nil
			})

		}
		err := eg.Wait()

		if err != nil {
			return nil, err
		}
		return amqp.NewPublisher(conHolders), nil
	}
	return nil, fmt.Errorf("Consumer cant open")
}
