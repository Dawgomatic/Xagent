// OneBot channel -- REMOVED for security (Chinese QQ bot protocol)
// This stub satisfies compile-time references. The channel will never initialize.

package channels

import (
	"context"
	"errors"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/config"
)

type OneBotChannel struct {
	*BaseChannel
}

func NewOneBotChannel(_ config.OneBotConfig, _ *bus.MessageBus) (*OneBotChannel, error) {
	return nil, errors.New("OneBot channel has been removed (Chinese QQ protocol)")
}

func (o *OneBotChannel) Start(_ context.Context) error {
	return errors.New("OneBot channel has been removed")
}

func (o *OneBotChannel) Stop(_ context.Context) error { return nil }

func (o *OneBotChannel) Send(_ context.Context, _ bus.OutboundMessage) error {
	return errors.New("OneBot channel has been removed")
}

func (o *OneBotChannel) IsRunning() bool { return false }
