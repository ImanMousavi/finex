package matching

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDepth(t *testing.T) {
	depth := NewDepth("BTC/CNY", &Notification{})

	assert.Equal(t, int64(1), 16)
	assert.Equal(t, "BTC/CNY", depth.Symbol)
}
