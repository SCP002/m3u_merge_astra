package copier

import (
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
)

// TDeep returns deep copy of <inp>, failing the test <t> if copier fails
func TDeep[V any](t *testing.T, inp V) V {
	out, err := deep(inp)
	assert.NoError(t, err, "should copy the source")
	return out
}

// PDeep returns deep copy of <inp>, panicking if copier fails
func PDeep[T any](inp T) T {
	out, err := deep(inp)
	if err != nil {
		panic(err)
	}
	return out
}

// deep returns deep copy of <inp>
func deep[T any](inp T) (out T, err error) {
	err = copier.CopyWithOption(&out, &inp, copier.Option{DeepCopy: true, IgnoreEmpty: true})
	err = errors.Wrap(err, "Copier")
	return
}
