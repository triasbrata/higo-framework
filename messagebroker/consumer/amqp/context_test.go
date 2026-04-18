package amqp

import (
	"context"
	"testing"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

func Test_contextAmqp_populateContext(t *testing.T) {
	type fields struct {
		ctx             context.Context
		body            []byte
		header          map[string]interface{}
		stack           *amqpStack
		ciStackHandler  int64
		lenStackHandler int64
	}
	type args struct {
		ctx         context.Context
		msgDelivery amqp091.Delivery
		stack       amqpStack
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &contextAmqp{
				ctx:             tt.fields.ctx,
				body:            tt.fields.body,
				header:          tt.fields.header,
				stack:           tt.fields.stack,
				ciStackHandler:  tt.fields.ciStackHandler,
				lenStackHandler: tt.fields.lenStackHandler,
			}
			c.populateContext(tt.args.ctx, tt.args.msgDelivery, tt.args.stack)
		})
	}
}
