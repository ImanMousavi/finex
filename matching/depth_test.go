package matching

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDepth(t *testing.T) {
	depth := NewDepth("BTC/CNY", &Notification{}, 1)

	assert.Equal(t, int64(1), depth.Scale)
	assert.Equal(t, "BTC/CNY", depth.Symbol)
}
