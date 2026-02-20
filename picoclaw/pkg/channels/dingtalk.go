// DingTalk channel -- REMOVED for security (Chinese service: Alibaba/DingTalk)
// This stub satisfies compile-time references. The channel will never initialize.

package channels

import (
	"context"
	"errors"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

type DingTalkChannel struct {
	*BaseChannel
}

func NewDingTalkChannel(_ config.DingTalkConfig, _ *bus.MessageBus) (*DingTalkChannel, error) {
	return nil, errors.New("DingTalk channel has been removed (Chinese service)")
}

func (d *DingTalkChannel) Start(_ context.Context) error {
	return errors.New("DingTalk channel has been removed")
}

func (d *DingTalkChannel) Stop(_ context.Context) error { return nil }

func (d *DingTalkChannel) Send(_ context.Context, _ bus.OutboundMessage) error {
	return errors.New("DingTalk channel has been removed")
}

func (d *DingTalkChannel) IsRunning() bool { return false }
