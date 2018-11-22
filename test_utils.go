package stream

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

var precision = 9

func roundFloat(x float64, n int) float64 {
	unit := 5 * math.Pow10(-n-1)
	return math.Round(x/unit) * unit
}

// Approx asserts that two floats are approximately equal to each other,
// within 9 decimal points of precision.
func Approx(t *testing.T, x float64, y float64) {
	x = roundFloat(x, precision)
	y = roundFloat(y, precision)
	assert.Equal(t, x, y)
}

// TestData returns a Core struct with example data populated from pushes for testing purposes.
// You can also pass in a variety of metrics to subscribe them to the core during testing.
func TestData(metrics ...Metric) *Core {
	core, err := NewCore(&CoreConfig{
		Sums: map[int]bool{
			-1: true,
			0:  true,
			1:  true,
			2:  true,
			3:  true,
			4:  true,
		},
		Window: IntPtr(3),
	}, metrics...)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	for i := 1.; i < 5; i++ {
		core.Push(i)
	}

	core.Push(8)

	return core
}