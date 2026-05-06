package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"ingest-srv/internal/dryrun"
	dryrunRabbit "ingest-srv/internal/dryrun/delivery/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
	"github.com/smap-hcmut/shared-libs/go/tracing"
	"github.com/stretchr/testify/require"
)

func TestNewConsumer(t *testing.T) {
	conn := &fakeRabbitConn{}
	uc := &fakeDryrunConsumerUseCase{}

	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output Consumer
		err    error
	}{
		"success": {
			output: Consumer{l: testLogger(), conn: conn, dryrunUC: uc},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := NewConsumer(testLogger(), conn, uc)
			require.Equal(t, tc.output.conn, output.conn)
			require.Equal(t, tc.output.dryrunUC, output.dryrunUC)
			require.NotNil(t, output.l)
		})
	}
}

func TestConsume(t *testing.T) {
	errChannel := errors.New("channel")
	errDeclare := errors.New("declare")
	errConsume := errors.New("consume")

	tcs := map[string]struct {
		input struct {
			consumer Consumer
			ctx      context.Context
		}
		mock   struct{}
		output struct {
			workerCalled bool
			closed       bool
		}
		err error
	}{
		"missing_connection": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{consumer: Consumer{l: testLogger(), dryrunUC: &fakeDryrunConsumerUseCase{}}, ctx: context.Background()},
			err: errors.New("rabbitmq client is required"),
		},
		"missing_usecase": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{consumer: Consumer{l: testLogger(), conn: &fakeRabbitConn{}}, ctx: context.Background()},
			err: errors.New("dryrun consumer usecase is required"),
		},
		"channel_error": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{
				consumer: Consumer{l: testLogger(), conn: &fakeRabbitConn{channelErr: errChannel}, dryrunUC: &fakeDryrunConsumerUseCase{}},
				ctx:      context.Background(),
			},
			err: errChannel,
		},
		"queue_declare_error": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{
				consumer: Consumer{
					l:        testLogger(),
					conn:     &fakeRabbitConn{channels: []*fakeRabbitChannel{{queueDeclareErr: errDeclare}}},
					dryrunUC: &fakeDryrunConsumerUseCase{},
				},
				ctx: context.Background(),
			},
			err: errDeclare,
		},
		"consume_error": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{
				consumer: Consumer{
					l:        testLogger(),
					conn:     &fakeRabbitConn{channels: []*fakeRabbitChannel{{consumeErr: errConsume}}},
					dryrunUC: &fakeDryrunConsumerUseCase{},
				},
				ctx: context.Background(),
			},
			err: errConsume,
		},
		"context_done": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{
				consumer: Consumer{
					l:        testLogger(),
					conn:     &fakeRabbitConn{channels: []*fakeRabbitChannel{{msgs: make(chan amqp.Delivery)}}},
					dryrunUC: &fakeDryrunConsumerUseCase{},
				},
				ctx: canceledContext(),
			},
			output: struct {
				workerCalled bool
				closed       bool
			}{closed: true},
		},
		"message_channel_closed": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{
				consumer: Consumer{
					l:        testLogger(),
					conn:     &fakeRabbitConn{channels: []*fakeRabbitChannel{{msgs: closedDeliveries()}}},
					dryrunUC: &fakeDryrunConsumerUseCase{},
				},
				ctx: context.Background(),
			},
			output: struct {
				workerCalled bool
				closed       bool
			}{closed: true},
		},
		"worker_called": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{
				consumer: Consumer{
					l:        testLogger(),
					conn:     &fakeRabbitConn{channels: []*fakeRabbitChannel{{msgs: oneDelivery()}}},
					dryrunUC: &fakeDryrunConsumerUseCase{},
				},
				ctx: context.Background(),
			},
			output: struct {
				workerCalled bool
				closed       bool
			}{workerCalled: true, closed: true},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(tc.input.ctx)
			defer cancel()

			workerCalled := false
			err := tc.input.consumer.consume(ctx, dryrunRabbit.IngestDryrunCompletionsQueue, "consumer", func(amqp.Delivery) {
				workerCalled = true
				cancel()
			})

			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.output.workerCalled, workerCalled)
			if tc.output.closed {
				require.True(t, tc.input.consumer.conn.(*fakeRabbitConn).channels[0].closed)
			}
		})
	}
}

func TestConsumerConsume(t *testing.T) {
	tcs := map[string]struct {
		input struct {
			consumer Consumer
			ctx      context.Context
		}
		mock   struct{}
		output struct{}
		err    error
	}{
		"delegates_to_completion_queue": {
			input: struct {
				consumer Consumer
				ctx      context.Context
			}{
				consumer: Consumer{
					l:        testLogger(),
					conn:     &fakeRabbitConn{channels: []*fakeRabbitChannel{{msgs: closedDeliveries()}}},
					dryrunUC: &fakeDryrunConsumerUseCase{},
				},
				ctx: context.Background(),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, tc.input.consumer.Consume(tc.input.ctx))
			ch := tc.input.consumer.conn.(*fakeRabbitConn).channels[0]
			require.Equal(t, dryrunRabbit.IngestDryrunCompletionsQueue.Name, ch.queue.Name)
			require.Equal(t, dryrunRabbit.IngestDryrunCompletionsConsumerName, ch.consume.Consumer)
		})
	}
}

func TestCatchPanic(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output struct{}
		err    error
	}{
		"recovers": {},
		"no_panic": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				defer catchPanic()
				if name == "recovers" {
					panic("boom")
				}
			})
		})
	}
}

func TestHandleCompletionWorker(t *testing.T) {
	errGeneric := errors.New("generic")
	itemCount := 3

	message := dryrunRabbit.CompletionMessage{
		TaskID:        "task-1",
		Status:        "success",
		CompletedAt:   "2026-05-06T10:00:00Z",
		StorageBucket: "bucket",
		StoragePath:   "path",
		BatchID:       "batch-1",
		Checksum:      "checksum",
		ItemCount:     &itemCount,
		Metadata:      map[string]interface{}{"platform": "youtube"},
	}
	body, err := json.Marshal(message)
	require.NoError(t, err)

	tcs := map[string]struct {
		input struct {
			body    []byte
			headers amqp.Table
		}
		mock struct {
			err error
		}
		output struct {
			acked    bool
			nacked   bool
			requeued bool
			input    dryrun.HandleCompletionInput
		}
		err error
	}{
		"invalid_json_ack": {
			input: struct {
				body    []byte
				headers amqp.Table
			}{body: []byte("{")},
			output: struct {
				acked    bool
				nacked   bool
				requeued bool
				input    dryrun.HandleCompletionInput
			}{acked: true},
		},
		"completion_not_found_ack": {
			input: struct {
				body    []byte
				headers amqp.Table
			}{body: body},
			mock: struct{ err error }{err: dryrun.ErrCompletionTaskNotFound},
			output: struct {
				acked    bool
				nacked   bool
				requeued bool
				input    dryrun.HandleCompletionInput
			}{acked: true, input: message.ToHandleCompletionInput()},
		},
		"invalid_completion_ack": {
			input: struct {
				body    []byte
				headers amqp.Table
			}{body: body},
			mock: struct{ err error }{err: dryrun.ErrInvalidCompletionInput},
			output: struct {
				acked    bool
				nacked   bool
				requeued bool
				input    dryrun.HandleCompletionInput
			}{acked: true, input: message.ToHandleCompletionInput()},
		},
		"generic_error_nack": {
			input: struct {
				body    []byte
				headers amqp.Table
			}{body: body},
			mock: struct{ err error }{err: errGeneric},
			output: struct {
				acked    bool
				nacked   bool
				requeued bool
				input    dryrun.HandleCompletionInput
			}{nacked: true, requeued: true, input: message.ToHandleCompletionInput()},
		},
		"success_ack_with_trace": {
			input: struct {
				body    []byte
				headers amqp.Table
			}{body: body, headers: amqp.Table{tracing.TraceIDHeader: "trace-1"}},
			output: struct {
				acked    bool
				nacked   bool
				requeued bool
				input    dryrun.HandleCompletionInput
			}{acked: true, input: message.ToHandleCompletionInput()},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ack := &fakeAcknowledger{}
			uc := &fakeDryrunConsumerUseCase{err: tc.mock.err}
			consumer := Consumer{l: testLogger(), dryrunUC: uc}

			consumer.handleCompletionWorker(amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  1,
				Headers:      tc.input.headers,
				Body:         tc.input.body,
			})

			require.Equal(t, tc.output.acked, ack.acked)
			require.Equal(t, tc.output.nacked, ack.nacked)
			require.Equal(t, tc.output.requeued, ack.requeued)
			if tc.input.body[0] != '{' || string(tc.input.body) != "{" {
				require.Equal(t, tc.output.input, uc.input)
			}
		})
	}
}

func testLogger() log.Logger {
	return log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func closedDeliveries() <-chan amqp.Delivery {
	msgs := make(chan amqp.Delivery)
	close(msgs)
	return msgs
}

func oneDelivery() <-chan amqp.Delivery {
	msgs := make(chan amqp.Delivery, 1)
	msgs <- amqp.Delivery{}
	return msgs
}

type fakeDryrunConsumerUseCase struct {
	input dryrun.HandleCompletionInput
	err   error
}

func (f *fakeDryrunConsumerUseCase) HandleCompletion(_ context.Context, input dryrun.HandleCompletionInput) error {
	f.input = input
	return f.err
}

type fakeAcknowledger struct {
	acked    bool
	nacked   bool
	rejected bool
	requeued bool
}

func (f *fakeAcknowledger) Ack(uint64, bool) error {
	f.acked = true
	return nil
}

func (f *fakeAcknowledger) Nack(_ uint64, _ bool, requeue bool) error {
	f.nacked = true
	f.requeued = requeue
	return nil
}

func (f *fakeAcknowledger) Reject(_ uint64, requeue bool) error {
	f.rejected = true
	f.requeued = requeue
	return nil
}

type fakeRabbitConn struct {
	channels   []*fakeRabbitChannel
	channelErr error
	calls      int
}

func (f *fakeRabbitConn) Close()         {}
func (f *fakeRabbitConn) IsReady() bool  { return true }
func (f *fakeRabbitConn) IsClosed() bool { return false }
func (f *fakeRabbitConn) Channel() (rabbitmq.IChannel, error) {
	if f.channelErr != nil {
		return nil, f.channelErr
	}
	if f.calls >= len(f.channels) {
		f.channels = append(f.channels, &fakeRabbitChannel{})
	}
	ch := f.channels[f.calls]
	f.calls++
	return ch, nil
}
func (f *fakeRabbitConn) ChannelWithTrace(context.Context) (rabbitmq.IChannel, error) {
	return f.Channel()
}

type fakeRabbitChannel struct {
	queue           amqp.Queue
	consume         rabbitmq.ConsumeArgs
	msgs            <-chan amqp.Delivery
	queueDeclareErr error
	consumeErr      error
	closed          bool
}

func (f *fakeRabbitChannel) ExchangeDeclare(rabbitmq.ExchangeArgs) error { return nil }
func (f *fakeRabbitChannel) ExchangeDeclareWithTrace(context.Context, rabbitmq.ExchangeArgs) error {
	return nil
}
func (f *fakeRabbitChannel) QueueDeclare(queue rabbitmq.QueueArgs) (amqp.Queue, error) {
	if f.queueDeclareErr != nil {
		return amqp.Queue{}, f.queueDeclareErr
	}
	f.queue = amqp.Queue{Name: queue.Name}
	return f.queue, nil
}
func (f *fakeRabbitChannel) QueueDeclareWithTrace(_ context.Context, queue rabbitmq.QueueArgs) (amqp.Queue, error) {
	return f.QueueDeclare(queue)
}
func (f *fakeRabbitChannel) QueueBind(rabbitmq.QueueBindArgs) error { return nil }
func (f *fakeRabbitChannel) QueueBindWithTrace(context.Context, rabbitmq.QueueBindArgs) error {
	return nil
}
func (f *fakeRabbitChannel) Publish(context.Context, rabbitmq.PublishArgs) error { return nil }
func (f *fakeRabbitChannel) PublishWithTrace(context.Context, rabbitmq.PublishArgs) error {
	return nil
}
func (f *fakeRabbitChannel) Consume(consume rabbitmq.ConsumeArgs) (<-chan amqp.Delivery, error) {
	if f.consumeErr != nil {
		return nil, f.consumeErr
	}
	f.consume = consume
	if f.msgs == nil {
		f.msgs = closedDeliveries()
	}
	return f.msgs, nil
}
func (f *fakeRabbitChannel) ConsumeWithTrace(_ context.Context, consume rabbitmq.ConsumeArgs) (<-chan amqp.Delivery, error) {
	return f.Consume(consume)
}
func (f *fakeRabbitChannel) Close() error {
	f.closed = true
	return nil
}
func (f *fakeRabbitChannel) NotifyReconnect(receiver chan bool) <-chan bool { return receiver }
