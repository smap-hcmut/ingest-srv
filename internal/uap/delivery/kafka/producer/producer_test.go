package producer

import (
	"context"
	"errors"
	"testing"

	"ingest-srv/internal/uap"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output uap.Publisher
		err    error
	}{
		"success": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			got := New(testLogger(), &fakeKafkaProducer{})
			require.NotNil(t, got)
		})
	}
}

func TestPublish(t *testing.T) {
	errPublish := errors.New("publish")

	tcs := map[string]struct {
		input  uap.PublishUAPInput
		mock   *fakeKafkaProducer
		output struct{}
		err    error
	}{
		"nil_producer": {
			input: uap.PublishUAPInput{Record: uap.UAPRecord{Identity: uap.UAPIdentity{UAPID: "uap-1"}}},
		},
		"success": {
			input: uap.PublishUAPInput{Record: uap.UAPRecord{Identity: uap.UAPIdentity{UAPID: " uap-1 "}}},
			mock:  &fakeKafkaProducer{},
		},
		"marshal_error": {
			input: uap.PublishUAPInput{Record: uap.UAPRecord{Identity: uap.UAPIdentity{UAPID: "uap-1"}, PlatformMeta: map[string]interface{}{"bad": func() {}}}},
			mock:  &fakeKafkaProducer{},
			err:   errors.New("json"),
		},
		"publish_error": {
			input: uap.PublishUAPInput{Record: uap.UAPRecord{Identity: uap.UAPIdentity{UAPID: "uap-1"}}},
			mock:  &fakeKafkaProducer{publishErr: errPublish},
			err:   errPublish,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p := &publisher{logger: testLogger()}
			if tc.mock != nil {
				p.producer = tc.mock
			}

			err := p.Publish(context.Background(), tc.input)

			if tc.err != nil {
				require.Error(t, err)
				if tc.err == errPublish {
					require.ErrorIs(t, err, tc.err)
				}
				return
			}
			require.NoError(t, err)
			if tc.mock != nil {
				require.Equal(t, []byte("uap-1"), tc.mock.key)
				require.NotEmpty(t, tc.mock.value)
			}
		})
	}
}

func TestTopic(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output string
		err    error
	}{
		"success": {output: "smap.collector.output"},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p := &publisher{}
			require.Equal(t, tc.output, p.Topic())
		})
	}
}

func TestClose(t *testing.T) {
	errClose := errors.New("close")

	tcs := map[string]struct {
		input  struct{}
		mock   *fakeKafkaProducer
		output struct{}
		err    error
	}{
		"nil_producer": {},
		"success":      {mock: &fakeKafkaProducer{}},
		"close_error":  {mock: &fakeKafkaProducer{closeErr: errClose}, err: errClose},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p := &publisher{}
			if tc.mock != nil {
				p.producer = tc.mock
			}
			err := p.Close()
			require.ErrorIs(t, err, tc.err)
			if tc.mock != nil {
				require.True(t, tc.mock.closed)
			}
		})
	}
}

func testLogger() log.Logger {
	return log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
}

type fakeKafkaProducer struct {
	key        []byte
	value      []byte
	publishErr error
	closeErr   error
	closed     bool
}

func (f *fakeKafkaProducer) Publish(key, value []byte) error {
	f.key = key
	f.value = value
	return f.publishErr
}

func (f *fakeKafkaProducer) PublishWithContext(context.Context, []byte, []byte) error {
	return nil
}

func (f *fakeKafkaProducer) Close() error {
	f.closed = true
	return f.closeErr
}

func (f *fakeKafkaProducer) HealthCheck() error { return nil }
