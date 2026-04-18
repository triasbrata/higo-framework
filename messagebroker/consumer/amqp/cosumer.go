package amqp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/triasbrata/higo/messagebroker/consumer"
	"github.com/triasbrata/higo/messagebroker/manager"
	"github.com/triasbrata/higo/messagebroker/manager/connections"
	"golang.org/x/sync/errgroup"
)

type amqpStack struct {
	queueName    string
	consumerName string
	topology     consumer.AmqpTopologyConsumer
	handlers     []consumer.ConsumeHandler
}
type csmr struct {
	man              manager.Manager[connections.ConnectionAMQP]
	restartTime      time.Duration
	mut              sync.Mutex
	stack            []amqpStack
	globalMiddleware []consumer.ConsumeHandler
	ctxPool          sync.Pool
	chanErr          chan error
	chanOk           chan struct{}
}

// Status implements consumer.Consumer.
func (c *csmr) Status() (ok chan struct{}, err chan error) {
	return c.chanOk, c.chanErr
}

// Consume implements consumer.Consumer.
func (c *csmr) Consume(queueName string, topology consumer.ConsumerTopology, handlers ...consumer.ConsumeHandler) consumer.ConsumerBuilder {
	c.mut.Lock()
	defer c.mut.Unlock()
	if len(handlers) == 0 {
		slog.Warn("cant register queue", slog.String("queueName", queueName))
		return c
	}
	c.stack = append(c.stack, amqpStack{
		queueName: queueName,
		topology:  topology().Amqp,
		handlers:  handlers,
	})

	return c
}

// SimpleConsume implements consumer.Consumer.
func (c *csmr) SimpleConsume(queueName string, handlers ...consumer.ConsumeHandler) consumer.ConsumerBuilder {
	c.mut.Lock()
	defer c.mut.Unlock()
	if len(handlers) == 0 {
		slog.Warn("cant register queue", slog.String("queueName", queueName))
		return c
	}
	c.stack = append(c.stack, amqpStack{
		queueName: queueName,
		topology:  consumer.AmqpTopologyConsumer{},
		handlers:  handlers,
	})
	return c
}

// Use implements consumer.Consumer.
func (c *csmr) Use(handlers ...consumer.ConsumeHandler) consumer.ConsumerBuilder {
	c.mut.Lock()
	defer c.mut.Unlock()
	if len(handlers) == 0 {
		slog.Warn("no global middleware was registered")
		return c
	}

	c.globalMiddleware = append(c.globalMiddleware, handlers...)
	return c
}
func (c *csmr) Start(ctx context.Context) error {
	eg, gctx := errgroup.WithContext(ctx)
	chanStack := make(chan amqpStack, len(c.stack))
	eg.Go(func() error {
		defer close(chanStack)
		compiledStack := make([]amqpStack, 0, len(c.stack))
		for _, stack := range c.stack {
			stack.handlers = append(c.globalMiddleware, stack.handlers...)
			fmt.Printf("stack.handlers: %v\n", stack.handlers)
			compiledStack = append(compiledStack, stack)
		}
		for range c.man.Ready() {
			slog.InfoContext(ctx, "connection ready, time to create consumer topology")
			if len(c.stack) == 0 {
				return fmt.Errorf("no routing consumer was define")
			}
			select {
			case c.chanOk <- struct{}{}:
			default:
			}

			for _, stack := range compiledStack {
				slog.InfoContext(ctx, "create consumer topology", slog.Any("stackQueue", stack.queueName))
				chanStack <- stack
			}

		}
		return nil
	})
	for stack := range chanStack {
		slog.Info("start new consumer", slog.Any("stackQueue", stack.queueName))

		eg.Go(func() error {
			err := c.buildAndConsume(gctx, stack)
			if err == nil {
				slog.InfoContext(ctx, fmt.Sprintf("will restart consumer in %s", c.restartTime))
				time.Sleep(c.restartTime)
				chanStack <- stack
			}
			return err
		})
	}
	err := eg.Wait()
	if err != nil {
		select {
		case c.chanErr <- err:
		default:
		}
		return err
	}
	return nil
}

func (c *csmr) buildAndConsume(gctx context.Context, stack amqpStack) error {
	err := c.buildTopology(gctx, stack)
	if err != nil && errors.Is(err, amqp091.ErrClosed) {
		slog.ErrorContext(gctx, "got error when define the topology", slog.Any("err", err), slog.Any("stackQueue", stack.queueName))

		return nil
	}
	if err != nil {
		return fmt.Errorf("err chan stack: %w", err)
	}
	err = c.startConsuming(gctx, stack)
	if err != nil && (errors.Is(err, amqp091.ErrClosed) || err.Error() == "delivery was closed") {
		slog.ErrorContext(gctx, "got error when consume", slog.Any("err", err), slog.Any("stackQueue", stack.queueName))
		return nil
	}

	if err != nil {
		return fmt.Errorf("err chan stack: %w", err)
	}
	return nil
}

func (c *csmr) startConsuming(gctx context.Context, stack amqpStack) error {
	ch, err := c.man.GetCon().Channel()
	if err != nil {
		return fmt.Errorf("error when open channel for %s: %w", stack.queueName, err)
	}
	defer ch.Close()
	closeNotif := ch.NotifyClose(make(chan *amqp091.Error))
	err = ch.Qos(int(stack.topology.PrefetchCount), 0, false)
	if err != nil {
		return fmt.Errorf("error when set qos %s: %w", stack.queueName, err)
	}
	del, err := ch.ConsumeWithContext(gctx,
		stack.queueName,
		stack.consumerName,
		stack.topology.AutoAck.Value(),
		stack.topology.Exclusive.Value(),
		stack.topology.NoLocal.Value(),
		stack.topology.NoWait.Value(),
		stack.topology.Args)
	if err != nil {
		return fmt.Errorf("error when try to consume %s: %w", stack.queueName, err)
	}
	//select
	for {
		select {
		case err := <-closeNotif:
			return err
		case msgDelivery, ok := <-del:
			if !ok {
				return fmt.Errorf("delivery was closed")
			}
			attrs := []any{
				slog.String("msg", string(msgDelivery.Body)),
				slog.String("tag", msgDelivery.ConsumerTag),
			}
			go c.processMessage(gctx, attrs, stack, msgDelivery)
		}
	}
}

func (c *csmr) processMessage(gctx context.Context, attrs []any, stack amqpStack, msgDelivery amqp091.Delivery) {
	consumerCtx := c.ctxPool.Get().(*contextAmqp)
	var err error
	defer c.postProcesMessage(gctx, consumerCtx, err, attrs, stack, msgDelivery)
	consumerCtx.populateContext(gctx, msgDelivery, stack)
	err = consumerCtx.Next()
	if err != nil {
		return
	}
}

func (c *csmr) postProcesMessage(gctx context.Context, ctx consumer.CtxConsumer, errHandler error, attrs []any, stack amqpStack, msgDelivery amqp091.Delivery) {
	c.ctxPool.Put(ctx) // release back the context
	if errHandler != nil {
		slog.ErrorContext(gctx, "Failed when consume", append(attrs, slog.Any("err", errHandler))...)
	}
	if errHandler != nil && !stack.topology.AutoAck.Value() && msgDelivery.Acknowledger != nil {
		errAck := msgDelivery.Acknowledger.Reject(msgDelivery.DeliveryTag, false)
		if errAck != nil {
			slog.ErrorContext(gctx, "Fail to reject", append(attrs, slog.Any("err", errAck))...)
		}
		return
	}
	if !stack.topology.AutoAck.Value() && msgDelivery.Acknowledger != nil {
		errAck := msgDelivery.Acknowledger.Ack(msgDelivery.DeliveryTag, false)
		if errAck != nil {
			slog.ErrorContext(gctx, "Fail to reject", append(attrs, slog.Any("err", errAck))...)
		}
	}
}

func (c *csmr) buildTopology(gctx context.Context, stack amqpStack) (err error) {
	var ch connections.ChannelAMQP
	ch, err = c.man.GetCon().Channel()
	if err != nil {
		return fmt.Errorf("error when open channel buildTopology: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(stack.queueName,
		stack.topology.Durable.Value(),
		stack.topology.AutoDelete.Value(),
		stack.topology.Exclusive.Value(),
		stack.topology.NoWait.Value(),
		stack.topology.Args)
	if err != nil {
		return fmt.Errorf("error when QueueDeclare for %s: %w", stack.queueName, err)
	}
	if stack.topology.BindExchange != nil {
		if stack.topology.BindExchange.Exchange != nil {
			err = ch.ExchangeDeclare(
				stack.topology.BindExchange.ExchangeName,
				stack.topology.BindExchange.Exchange.Kind,
				stack.topology.BindExchange.Exchange.Durable.Value(),
				stack.topology.BindExchange.Exchange.AutoDelete.Value(),
				stack.topology.BindExchange.Exchange.Internal.Value(),
				stack.topology.BindExchange.NoWait.Value(),
				stack.topology.BindExchange.Exchange.Args)
			if err != nil {
				return fmt.Errorf("error when ExchangeDeclare for %s: %w", stack.queueName, err)
			}
		}
		err = ch.QueueBind(stack.queueName,
			stack.topology.BindExchange.RoutingKey,
			stack.topology.BindExchange.ExchangeName,
			stack.topology.BindExchange.NoWait.Value(),
			stack.topology.BindExchange.Args)
		if err != nil {
			return fmt.Errorf("error when QueueBind for %s: %w", stack.queueName, err)
		}
	}
	return nil
}

func NewConsumer(conManager manager.Manager[connections.ConnectionAMQP], restartTime time.Duration) consumer.Consumer {
	return &csmr{
		man:              conManager,
		mut:              sync.Mutex{},
		restartTime:      restartTime,
		stack:            []amqpStack{},
		globalMiddleware: make([]consumer.ConsumeHandler, 0),
		chanErr:          make(chan error, 1),
		chanOk:           make(chan struct{}, 1),
		ctxPool: sync.Pool{
			New: func() any {
				return &contextAmqp{}
			},
		},
	}
}
