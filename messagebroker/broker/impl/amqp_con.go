package impl

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/triasbrata/higo/messagebroker/manager"
	"github.com/triasbrata/higo/messagebroker/manager/connections"
	"github.com/triasbrata/higo/messagebroker/manager/connections/wrap"
	"github.com/triasbrata/higo/messagebroker/manager/impl"
)

func (b *brk) openConnectionAmqp(ctx context.Context, subfix ...any) (manager.Manager[connections.ConnectionAMQP], error) {
	man := impl.NewManager()
	conProps := amqp091.NewConnectionProperties()
	conName := b.cfg.amqp.name + fmt.Sprint(subfix...)
	conProps.SetClientConnectionName(conName)
	dialConf := amqp091.Config{
		Vhost:      b.cfg.amqp.vhost,
		Properties: conProps,
		Locale:     b.cfg.amqp.locale,
		ChannelMax: b.cfg.amqp.chanMax,
		Heartbeat:  b.cfg.amqp.heartBeat,
	}
	con, err := amqp091.DialConfig(b.cfg.amqp.uri, dialConf)
	if err != nil {
		return nil, fmt.Errorf("got error when dialup: %w", err)
	}

	go b.watchConnectionAmqp(ctx, man, con, dialConf)
	return man, nil
}

func (b *brk) watchConnectionAmqp(ctx context.Context, man manager.Manager[connections.ConnectionAMQP], con *amqp091.Connection, dialConf amqp091.Config) {
	var err error
	for {
		if con.IsClosed() {
			slog.InfoContext(ctx, fmt.Sprintf("will reopen the connection on %v", b.restartTimer),
				slog.Any("retryCount", b.cfg.amqp.retryCounter.Load()),
				slog.String("connectionName", b.cfg.amqp.name),
			)
			time.Sleep(b.restartTimer)
			dialConf.Properties.SetClientConnectionName(fmt.Sprintf("%s-%v", b.cfg.amqp.name, b.cfg.amqp.retryCounter.Load()))
			con, err = amqp091.DialConfig(b.cfg.amqp.uri, dialConf)
			if err != nil {
				slog.ErrorContext(ctx, "fail dialing up the connection",
					slog.String("connectionName", b.cfg.amqp.name),
					slog.Any("err", err),
				)
				continue
			}
		}
		closeNotif := con.NotifyClose(make(chan *amqp091.Error))
		man.SetCon(&wrap.Connection{Connection: con})
		select {
		case <-ctx.Done():
			slog.Info("context Done")
			con.Close()
			return
		case err := <-closeNotif:
			slog.ErrorContext(ctx, "connection closed", slog.Any("err", err))
			b.cfg.amqp.retryCounter.Add(1)
		}
	}
}
