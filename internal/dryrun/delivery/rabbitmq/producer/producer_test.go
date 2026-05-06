package producer

import (
	"context"
	"errors"
	"testing"

	"ingest-srv/internal/dryrun"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/smap-hcmut/shared-libs/go/constants"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output Producer
		err    error
	}{
		"success": {},
	}
	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, New(testLogger(), &fakeRabbitConn{}))
		})
	}
}

func TestRun(t *testing.T) {
	errChannel := errors.New("channel")
	errQueue := errors.New("queue")
	errExchange := errors.New("exchange")
	errBind := errors.New("bind")

	tcs := map[string]struct {
		input  struct{}
		mock   *fakeRabbitConn
		output struct{}
		err    error
	}{
		"nil_conn": {},
		"success": {
			mock: &fakeRabbitConn{channels: []*fakeRabbitChannel{{}, {}, {}}},
		},
		"first_channel_error": {
			mock: &fakeRabbitConn{channelErr: errChannel},
			err:  errChannel,
		},
		"first_queue_error": {
			mock: &fakeRabbitConn{channels: []*fakeRabbitChannel{{queueErr: errQueue}}},
			err:  errQueue,
		},
		"second_exchange_error_closes_first": {
			mock: &fakeRabbitConn{channels: []*fakeRabbitChannel{{}, {exchangeErr: errExchange}}},
			err:  errExchange,
		},
		"third_bind_error_closes_previous": {
			mock: &fakeRabbitConn{channels: []*fakeRabbitChannel{{}, {}, {bindErr: errBind}}},
			err:  errBind,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p := &implProducer{l: testLogger()}
			if tc.mock != nil {
				p.conn = tc.mock
			}

			err := p.Run()

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil && tc.mock != nil {
				require.NotNil(t, p.tikTokTasksWriter)
				require.NotNil(t, p.facebookTasksWriter)
				require.NotNil(t, p.youtubeTasksWriter)
			}
		})
	}
}

func TestClose(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   []*fakeRabbitChannel
		output struct{}
		err    error
	}{
		"nil_writers": {},
		"success":     {mock: []*fakeRabbitChannel{{}, {}, {}}},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p := &implProducer{}
			if len(tc.mock) == 3 {
				p.tikTokTasksWriter = tc.mock[0]
				p.facebookTasksWriter = tc.mock[1]
				p.youtubeTasksWriter = tc.mock[2]
			}
			p.Close()
			for _, ch := range tc.mock {
				require.True(t, ch.closed)
			}
		})
	}
}

func TestPublishDispatch(t *testing.T) {
	errPublish := errors.New("publish")

	tcs := map[string]struct {
		input  dryrun.PublishDispatchInput
		mock   *fakeRabbitChannel
		output struct{}
		err    bool
	}{
		"success_tiktok": {
			input: dryrun.PublishDispatchInput{Queue: dryrun.QueueName(constants.QueueTikTokTasks), TaskID: "task-1", Action: dryrun.ActionNameFullFlow},
			mock:  &fakeRabbitChannel{},
		},
		"marshal_error": {
			input: dryrun.PublishDispatchInput{Queue: dryrun.QueueName(constants.QueueTikTokTasks), TaskID: "task-1", Action: dryrun.ActionNameFullFlow, Params: map[string]interface{}{"bad": func() {}}},
			mock:  &fakeRabbitChannel{},
			err:   true,
		},
		"unsupported_queue": {
			input: dryrun.PublishDispatchInput{Queue: "unknown", TaskID: "task-1", Action: dryrun.ActionNameFullFlow},
			err:   true,
		},
		"writer_missing": {
			input: dryrun.PublishDispatchInput{Queue: dryrun.QueueName(constants.QueueFacebookTasks), TaskID: "task-1", Action: dryrun.ActionNamePostDetail},
			err:   true,
		},
		"publish_error": {
			input: dryrun.PublishDispatchInput{Queue: dryrun.QueueName(constants.QueueTikTokTasks), TaskID: "task-1", Action: dryrun.ActionNameFullFlow},
			mock:  &fakeRabbitChannel{publishErr: errPublish},
			err:   true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p := &implProducer{tikTokTasksWriter: tc.mock}

			err := p.PublishDispatch(context.Background(), tc.input)

			require.Equal(t, tc.err, err != nil)
			if !tc.err {
				require.Equal(t, constants.ExchangeTikTokTasks, tc.mock.publish.Exchange)
				require.NotEmpty(t, tc.mock.publish.Msg.Body)
			}
		})
	}
}

func TestGetPublishRouteByQueue(t *testing.T) {
	tcs := map[string]struct {
		input  dryrun.QueueName
		mock   func(*implProducer)
		output string
		err    bool
	}{
		"tiktok": {
			input:  dryrun.QueueName(constants.QueueTikTokTasks),
			mock:   func(p *implProducer) { p.tikTokTasksWriter = &fakeRabbitChannel{} },
			output: constants.ExchangeTikTokTasks,
		},
		"facebook": {
			input:  dryrun.QueueName(constants.QueueFacebookTasks),
			mock:   func(p *implProducer) { p.facebookTasksWriter = &fakeRabbitChannel{} },
			output: constants.ExchangeFacebookTasks,
		},
		"youtube": {
			input:  dryrun.QueueName(constants.QueueYouTubeTasks),
			mock:   func(p *implProducer) { p.youtubeTasksWriter = &fakeRabbitChannel{} },
			output: constants.ExchangeYouTubeTasks,
		},
		"missing_tiktok":   {input: dryrun.QueueName(constants.QueueTikTokTasks), err: true},
		"missing_facebook": {input: dryrun.QueueName(constants.QueueFacebookTasks), err: true},
		"missing_youtube":  {input: dryrun.QueueName(constants.QueueYouTubeTasks), err: true},
		"unknown":          {input: "unknown", err: true},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p := &implProducer{}
			if tc.mock != nil {
				tc.mock(p)
			}
			_, exchange, _, err := p.getPublishRouteByQueue(tc.input)
			require.Equal(t, tc.err, err != nil)
			require.Equal(t, tc.output, exchange)
		})
	}
}

func TestMockProducerGenerated(t *testing.T) {
	errRun := errors.New("run")
	errPublish := errors.New("publish")
	input := dryrun.PublishDispatchInput{TaskID: "task-1"}

	tcs := map[string]struct {
		input  struct{}
		mock   func(*MockProducer)
		output struct{}
		err    error
	}{
		"return_helpers": {
			mock: func(m *MockProducer) {
				m.EXPECT().Close().Run(func() {}).Return()
				m.EXPECT().Run().Run(func() {}).Return(errRun)
				m.EXPECT().PublishDispatch(context.Background(), input).Run(func(context.Context, dryrun.PublishDispatchInput) {}).Return(errPublish)
			},
			err: errRun,
		},
		"run_and_return_helpers": {
			mock: func(m *MockProducer) {
				m.EXPECT().Close().RunAndReturn(func() {})
				m.EXPECT().Run().RunAndReturn(func() error { return nil })
				m.EXPECT().PublishDispatch(context.Background(), input).RunAndReturn(func(context.Context, dryrun.PublishDispatchInput) error { return nil })
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			m := NewMockProducer(t)
			tc.mock(m)

			m.Close()
			err := m.Run()
			require.ErrorIs(t, err, tc.err)
			err = m.PublishDispatch(context.Background(), input)
			if tc.err != nil {
				require.ErrorIs(t, err, errPublish)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("panic_without_return", func(t *testing.T) {
		m := &MockProducer{}
		m.On("Run")
		require.Panics(t, func() { _ = m.Run() })

		m = &MockProducer{}
		m.On("PublishDispatch", context.Background(), input)
		require.Panics(t, func() { _ = m.PublishDispatch(context.Background(), input) })
	})

	t.Run("direct_function_returns", func(t *testing.T) {
		m := &MockProducer{}
		m.On("Run").Return(func() error { return nil })
		m.On("PublishDispatch", context.Background(), input).Return(func(context.Context, dryrun.PublishDispatchInput) error { return nil })
		require.NoError(t, m.Run())
		require.NoError(t, m.PublishDispatch(context.Background(), input))
	})
}

func testLogger() log.Logger {
	return log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
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
	queueErr    error
	exchangeErr error
	bindErr     error
	publishErr  error
	closed      bool
	publish     rabbitmq.PublishArgs
}

func (f *fakeRabbitChannel) ExchangeDeclare(rabbitmq.ExchangeArgs) error { return f.exchangeErr }
func (f *fakeRabbitChannel) ExchangeDeclareWithTrace(context.Context, rabbitmq.ExchangeArgs) error {
	return f.exchangeErr
}
func (f *fakeRabbitChannel) QueueDeclare(queue rabbitmq.QueueArgs) (amqp.Queue, error) {
	return amqp.Queue{Name: queue.Name}, f.queueErr
}
func (f *fakeRabbitChannel) QueueDeclareWithTrace(context.Context, rabbitmq.QueueArgs) (amqp.Queue, error) {
	return amqp.Queue{}, nil
}
func (f *fakeRabbitChannel) QueueBind(rabbitmq.QueueBindArgs) error { return f.bindErr }
func (f *fakeRabbitChannel) QueueBindWithTrace(context.Context, rabbitmq.QueueBindArgs) error {
	return f.bindErr
}
func (f *fakeRabbitChannel) Publish(_ context.Context, publish rabbitmq.PublishArgs) error {
	f.publish = publish
	return f.publishErr
}
func (f *fakeRabbitChannel) PublishWithTrace(context.Context, rabbitmq.PublishArgs) error {
	return nil
}
func (f *fakeRabbitChannel) Consume(rabbitmq.ConsumeArgs) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (f *fakeRabbitChannel) ConsumeWithTrace(context.Context, rabbitmq.ConsumeArgs) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (f *fakeRabbitChannel) Close() error {
	f.closed = true
	return nil
}
func (f *fakeRabbitChannel) NotifyReconnect(receiver chan bool) <-chan bool { return receiver }
