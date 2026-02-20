// Feishu/Lark channel -- REMOVED for security (Chinese service: ByteDance/Feishu)
// Stub for 32-bit architectures.

//go:build !amd64 && !arm64 && !riscv64 && !mips64 && !ppc64

package channels

import (
	"context"
	"errors"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func NewFeishuChannel(_ config.FeishuConfig, _ *bus.MessageBus) (*FeishuChannel, error) {
	return nil, errors.New("Feishu channel has been removed (Chinese service)")
}
