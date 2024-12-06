package joint

import (
	"math"
	"sync"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/pkg/errors"

	"github.com/alexander-yu/stream"
	mathutil "github.com/alexander-yu/stream/util/math"
)

// Core is a struct that stores fundamental information for multivariate moments of a stream.
type Core struct {
	mux     sync.RWMutex
	means   []float64
	tuples  []Tuple
	sums    map[uint64]float64
	newSums map[uint64]float64
	count   int
	window  int
	decay   *float64
	queue   *queue.RingBuffer
}

// Init sets a CoreWrapper up with a core for consuming.
func Init(wrapper CoreWrapper) error {
	config := wrapper.Config()
	core, err := NewCore(config)
	if err != nil {
		return errors.Wrap(err, "error creating Core")
	}

	wrapper.SetCore(core)
	return nil
}

// NewCore instantiates a Core struct based on a provided config.
func NewCore(config *CoreConfig) (*Core, error) {
	// set defaults for any remaining unset fields
	config = setConfigDefaults(config)

	// validate config
	err := validateConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "error validating config")
	}

	if config.Vars == nil && len(config.Sums) > 0 {
		config.Vars = stream.IntPtr(len(config.Sums[0]))
	}

	// initialize and create Core
	c := &Core{}
	c.window = *config.Window
	c.decay = config.Decay

	c.sums = map[uint64]float64{}
	c.newSums = map[uint64]float64{}
	for _, tuple := range config.Sums {
		_ = iter(tuple, false, func(xs ...int) error {
			c.sums[Tuple(xs).hash()] = 0
			return nil
		})
		_ = iter(tuple, false, func(xs ...int) error {
			c.newSums[Tuple(xs).hash()] = 0
			return nil
		})
	}

	c.tuples = config.Sums
	c.means = make([]float64, *config.Vars)
	c.queue = queue.NewRingBuffer(uint64(c.window))

	return c, nil
}

// Push adds a new value for a Core object to consume.
func (c *Core) Push(xs ...float64) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.UnsafePush(xs...)
}

// UnsafePush adds a new value for a Core object to consume,
// but does not lock. This should only be used if the user
// plans to make use of the Lock()/Unlock() Core methods.
func (c *Core) UnsafePush(xs ...float64) error {
	if len(xs) != len(c.means) {
		return errors.Errorf(
			"tried to push %d values when core is tracking %d variables",
			len(xs),
			len(c.means),
		)
	}

	if c.window != 0 {
		if c.queue.Len() == uint64(c.window) {
			tail, err := c.queue.Get()
			if err != nil {
				return errors.Wrap(err, "error popping item from queue")
			}

			err = c.remove(tail.([]float64)...)
			if err != nil {
				return errors.Wrapf(err, "error removing %v from sums", xs)
			}
		}

		err := c.queue.Put(xs)
		if err != nil {
			return errors.Wrapf(err, "error pushing %v to queue", xs)
		}
	}

	var err error
	if c.decay == nil {
		err = c.add(xs...)
	} else {
		err = c.addDecay(xs...)
	}
	if err != nil {
		return errors.Wrapf(err, "error adding %v to sums", xs)
	}

	return nil
}

// add updates the mean, count, and joint centralized power sums in an efficient
// and stable (numerically speaking) way, which allows for more accurate reporting
// of moments. See the following paper for details on the algorithm used:
// P. Pebay, T. B. Terriberry, H. Kolla, J. Bennett, Numerically stable, scalable
// formulas for parallel and online computation of higher-order multivariate central
// moments with arbitrary weights, Computational Statistics 31 (2016) 1305–1325.
func (c *Core) add(xs ...float64) error {
	c.count++
	count := float64(c.count)

	delta := make([]float64, len(c.means))
	for i, x := range xs {
		delta[i] = x - c.means[i]
		c.means[i] += delta[i] / count
	}

	for _, tuple := range c.tuples {
		err := iter(tuple, true, func(xs ...int) error {
			a := Tuple(xs)
			hash := a.hash()
			c.newSums[hash] = 0
			return iter(a, false, func(xs ...int) error {
				b := Tuple(xs)

				deltaPow, err := pow(delta, b)
				if err != nil {
					return err
				}

				abs := b.abs()
				if abs == 0 {
					c.newSums[hash] += c.sums[hash]
				} else if b.eq(a) {
					coeff := (count - 1) / math.Pow(count, float64(abs)) *
						(math.Pow(count-1, float64(abs-1)) + float64(mathutil.Sign(abs)))
					c.newSums[hash] += coeff * deltaPow
				} else {
					multinomial, err := multinom(a, b)
					if err != nil {
						return err
					}

					diff, err := sub(a, b)
					if err != nil {
						return err
					}

					c.newSums[hash] += float64(multinomial*mathutil.Sign(abs)) /
						math.Pow(count, float64(abs)) * deltaPow * c.sums[diff.hash()]
				}

				return nil
			})
		})
		if err != nil {
			return errors.Wrapf(err, "error adding %v to sums for tuple %v", xs, tuple)
		}
	}

	for hash, sum := range c.newSums {
		c.sums[hash] = sum
	}

	return nil
}

// addDecay updates the mean, count, and joint centralized power sums (with exponential decay)
// in an efficient and stable (numerically speaking) way, which allows for more accurate
// reporting of moments. See the following paper for details on the algorithm used:
// P. Pebay, T. B. Terriberry, H. Kolla, J. Bennett, Numerically stable, scalable
// formulas for parallel and online computation of higher-order multivariate central
// moments with arbitrary weights, Computational Statistics 31 (2016) 1305–1325.
func (c *Core) addDecay(xs ...float64) error {
	c.count++

	var decay float64
	if c.count == 1 {
		decay = 1
	} else {
		decay = *c.decay
	}

	delta := make([]float64, len(c.means))
	for i, x := range xs {
		delta[i] = x - c.means[i]
		c.means[i] += decay * delta[i]
	}

	for _, tuple := range c.tuples {
		err := iter(tuple, true, func(xs ...int) error {
			a := Tuple(xs)
			hash := a.hash()
			c.newSums[hash] = 0
			return iter(a, false, func(xs ...int) error {
				b := Tuple(xs)

				deltaPow, err := pow(delta, b)
				if err != nil {
					return err
				}

				abs := b.abs()
				if abs == 0 {
					c.newSums[hash] += (1 - decay) * c.sums[hash]
				} else if b.eq(a) {
					coeff := (1-decay)*math.Pow(-decay, float64(abs)) + decay*math.Pow(1-decay, float64(abs))
					c.newSums[hash] += coeff * deltaPow
				} else {
					multinomial, err := multinom(a, b)
					if err != nil {
						return err
					}

					diff, err := sub(a, b)
					if err != nil {
						return err
					}

					c.newSums[hash] += float64(multinomial*mathutil.Sign(abs)) *
						math.Pow(decay, float64(abs)) * deltaPow *
						(1 - decay) * c.sums[diff.hash()]
				}

				return nil
			})
		})
		if err != nil {
			return errors.Wrapf(err, "error adding %v to sums for tuple %v", xs, tuple)
		}
	}

	for hash, sum := range c.newSums {
		c.sums[hash] = sum
	}

	return nil
}

// remove simply undoes the result of an add() call, and clears out the stats
// if we remove the last item of a window (only needed in the case where the
// window size is 1).
func (c *Core) remove(xs ...float64) error {
	c.count--
	if c.count > 0 {
		count := float64(c.count)

		delta := make([]float64, len(c.means))
		for i, x := range xs {
			c.means[i] -= (x - c.means[i]) / count
			delta[i] = x - c.means[i]
		}

		for _, tuple := range c.tuples {
			err := iter(tuple, false, func(xs ...int) error {
				a := Tuple(xs)
				hash := a.hash()
				c.newSums[hash] = 0
				return iter(a, false, func(xs ...int) error {
					b := Tuple(xs)

					deltaPow, err := pow(delta, b)
					if err != nil {
						return err
					}

					abs := b.abs()
					if abs == 0 {
						c.newSums[hash] += c.sums[hash]
					} else if b.eq(a) {
						coeff := count / math.Pow(count+1, float64(abs)) *
							(math.Pow(count, float64(abs-1)) + float64(mathutil.Sign(abs)))
						c.newSums[hash] -= coeff * deltaPow
					} else {
						multinomial, err := multinom(a, b)
						if err != nil {
							return err
						}

						diff, err := sub(a, b)
						if err != nil {
							return err
						}

						c.newSums[hash] -= float64(multinomial*mathutil.Sign(abs)) /
							math.Pow(count+1, float64(abs)) * deltaPow * c.newSums[diff.hash()]
					}

					return nil
				})
			})
			if err != nil {
				return errors.Wrapf(err, "error removing %v from sums for tuple %v", xs, tuple)
			}

			for hash, sum := range c.newSums {
				c.sums[hash] = sum
			}
		}
	} else {
		for i := range c.means {
			c.means[i] = 0
		}
		for hash := range c.sums {
			c.sums[hash] = 0
		}
		for hash := range c.newSums {
			c.newSums[hash] = 0
		}
	}

	return nil
}

// Count returns the number of values seen seen globally.
func (c *Core) Count() int {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.UnsafeCount()
}

// UnsafeCount returns the number of values seen seen globally,
// but does not lock. This should only be used if the user
// plans to make use of the [R]Lock()/[R]Unlock() Core methods.
func (c *Core) UnsafeCount() int {
	return c.count
}

// Mean returns the mean of values seen for a given variable.
func (c *Core) Mean(i int) (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.UnsafeMean(i)
}

// UnsafeMean returns the mean of values seen for a given variable,
// but does not lock. This should only be used if the user
// plans to make use of the [R]Lock()/[R]Unlock() Core methods.
func (c *Core) UnsafeMean(i int) (float64, error) {
	if c.count == 0 {
		return 0, errors.New("no values seen yet")
	}

	if i < 0 || i >= len(c.means) {
		return 0, errors.Errorf("%d is not a tracked variable", i)
	}

	return c.means[i], nil
}

// Sum returns the joint centralized sum of values seen for a provided
// exponent Tuple. In other words, for a Tuple m = (m_1, ..., m_k),
// this returns the sum of (x_i1 - μ_1)^m_1 * ... * (x_ik - μ_k)^m_k over
// all joint data points (x_i1, ..., x_ik).
func (c *Core) Sum(xs ...int) (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.UnsafeSum(xs...)
}

// UnsafeSum returns the joint centralized sum of values seen for a provided
// exponent Tuple, but does not lock. This should only be used if the user
// plans to make use of the [R]Lock()/[R]Unlock() Core methods.
func (c *Core) UnsafeSum(xs ...int) (float64, error) {
	if c.count == 0 {
		return 0, errors.New("no values seen yet")
	}

	sum, ok := c.sums[Tuple(xs).hash()]
	if !ok {
		return 0, errors.Errorf("%v is not a tracked power sum", xs)
	}

	return sum, nil
}

// Clear clears all stats being tracked.
func (c *Core) Clear() {
	c.mux.Lock()
	c.UnsafeClear()
	c.mux.Unlock()
}

// UnsafeClear clears all stats being tracked,
// but does not lock. This should only be used if the user
// plans to make use of the Lock()/Unlock() Core methods.
func (c *Core) UnsafeClear() {
	for i := range c.means {
		c.means[i] = 0
	}
	for hash := range c.sums {
		c.sums[hash] = 0
	}
	for hash := range c.newSums {
		c.newSums[hash] = 0
	}

	c.count = 0
	c.queue.Dispose()
	c.queue = queue.NewRingBuffer(uint64(c.window))
}

// RLock locks the Core internals for reading.
func (c *Core) RLock() {
	c.mux.RLock()
}

// RUnlock undoes a single RLock call.
func (c *Core) RUnlock() {
	c.mux.RUnlock()
}

// Lock locks the Core internals for writing.
func (c *Core) Lock() {
	c.mux.Lock()
}

// Unlock undoes a Lock call.
func (c *Core) Unlock() {
	c.mux.Unlock()
}
