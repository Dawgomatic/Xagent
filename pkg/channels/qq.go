// QQ channel -- REMOVED for security (Chinese service: Tencent/QQ)
// This stub satisfies compile-time references. The channel will never initialize.

package channels

import (
	"context"
	"errors"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

type QQChannel struct {
	*BaseChannel
}

func NewQQChannel(_ config.QQConfig, _ *bus.MessageBus) (*QQChannel, error) {
	return nil, errors.New("QQ channel has been removed (Chinese service)")
}

func (q *QQChannel) Start(_ context.Context) error {
	return errors.New("QQ channel has been removed")
}

func (q *QQChannel) Stop(_ context.Context) error { return nil }

func (q *QQChannel) Send(_ context.Context, _ bus.OutboundMessage) error {
	return errors.New("QQ channel has been removed")
}

func (q *QQChannel) IsRunning() bool { return false }
