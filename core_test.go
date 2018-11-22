package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPush(t *testing.T) {
	core := TestData()

	expectedSums := map[int]float64{
		-1: 17. / 24.,
		0:  3.,
		1:  15.,
		2:  89.,
		3:  603.,
		4:  4433.,
	}

	assert.Equal(t, len(expectedSums), len(core.sums))
	for k, expectedSum := range expectedSums {
		actualSum, ok := core.sums[k]
		require.True(t, ok)
		Approx(t, expectedSum, actualSum)
	}
}

func TestClear(t *testing.T) {
	core := TestData()
	core.Clear()

	expectedSums := map[int]float64{
		-1: 0,
		0:  0,
		1:  0,
		2:  0,
		3:  0,
		4:  0,
	}
	assert.Equal(t, expectedSums, core.sums)
}

func TestMin(t *testing.T) {
	core := TestData()
	Approx(t, 1, core.Min())
}

func TestMax(t *testing.T) {
	core := TestData()
	Approx(t, 8, core.Max())
}

func TestCount(t *testing.T) {
	core := TestData()
	assert.Equal(t, 5, core.Count())
}

func TestWindowCount(t *testing.T) {
	core := TestData()
	assert.Equal(t, 3, core.WindowCount())
}

func TestSum(t *testing.T) {
	t.Run("pass: Sum returns the correct sum", func(t *testing.T) {
		core := TestData()
		expectedSums := map[int]float64{
			-1: 17. / 24.,
			0:  3.,
			1:  15.,
			2:  89.,
			3:  603.,
			4:  4433.,
		}

		for i := -1; i <= 4; i++ {
			sum, err := core.Sum(i)
			require.Nil(t, err)
			Approx(t, expectedSums[i], sum)
		}
	})

	t.Run("fail: Sum fails if no elements consumed yet", func(t *testing.T) {
		core, err := NewCore(&CoreConfig{})
		require.NoError(t, err)

		_, err = core.Sum(1)
		assert.EqualError(t, err, "no values seen yet")
	})

	t.Run("fail: Sum fails for untracked power sum", func(t *testing.T) {
		core := TestData()

		_, err := core.Sum(10)
		assert.EqualError(t, err, "10 is not a tracked power sum")
	})
}