// Feishu/Lark channel -- REMOVED for security (Chinese service: ByteDance/Feishu)
// Stub for 32-bit architectures.

//go:build !amd64 && !arm64 && !riscv64 && !mips64 && !ppc64

package channels

import (
	"context"
	"errors"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/config"
)

func NewFeishuChannel(_ config.FeishuConfig, _ *bus.MessageBus) (*FeishuChannel, error) {
	return nil, errors.New("Feishu channel has been removed (Chinese service)")
}
