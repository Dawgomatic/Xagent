// Feishu/Lark channel -- REMOVED for security (Chinese service: ByteDance/Feishu)
// This stub satisfies compile-time references. The channel will never initialize.

//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package channels

import (
	"context"
	"errors"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/config"
)

type FeishuChannel struct {
	*BaseChannel
}

func NewFeishuChannel(_ config.FeishuConfig, _ *bus.MessageBus) (*FeishuChannel, error) {
	return nil, errors.New("Feishu channel has been removed (Chinese service)")
}

func (f *FeishuChannel) Start(_ context.Context) error {
	return errors.New("Feishu channel has been removed")
}

func (f *FeishuChannel) Stop(_ context.Context) error { return nil }

func (f *FeishuChannel) Send(_ context.Context, _ bus.OutboundMessage) error {
	return errors.New("Feishu channel has been removed")
}

func (f *FeishuChannel) IsRunning() bool { return false }
