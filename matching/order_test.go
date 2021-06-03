package matching

import (
	"testing"
	"unsafe"

	"github.com/cockroachdb/apd"
)

func TestOrderSize(t *testing.T) {
	o := Order{}
	t.Logf("sizeof: %db", unsafe.Sizeof(o))
}

func TestOrder_Pricing(t *testing.T) {
	t.Log(apd.New(12, 0).String())
}
