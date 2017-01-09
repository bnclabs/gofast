package main

import "math"
import "sync/atomic"

type averageInt64 struct {
	n      int64
	minval int64
	maxval int64
	sum    int64
	sumsq  uint64 // float64
	init   int64  // bool
}

func (av *averageInt64) add(sample int64) {
	atomic.AddInt64(&av.n, 1)
	atomic.AddInt64(&av.sum, sample)
	sumsq := math.Float64frombits(atomic.LoadUint64(&av.sumsq))
	f := float64(sample)
	sumsq += f * f
	atomic.StoreUint64(&av.sumsq, math.Float64bits(sumsq))

	init, minval := atomic.LoadInt64(&av.init), atomic.LoadInt64(&av.minval)
	if init == 0 || sample < minval {
		atomic.StoreInt64(&av.minval, sample)
		atomic.StoreInt64(&av.init, 1)
	}
	maxval := atomic.LoadInt64(&av.maxval)
	if maxval < sample {
		atomic.StoreInt64(&av.maxval, sample)
	}
}

func (av *averageInt64) min() int64 {
	return atomic.LoadInt64(&av.minval)
}

func (av *averageInt64) max() int64 {
	return atomic.LoadInt64(&av.maxval)
}

func (av *averageInt64) samples() int64 {
	return atomic.LoadInt64(&av.n)
}

func (av *averageInt64) total() int64 {
	return atomic.LoadInt64(&av.sum)
}

func (av *averageInt64) mean() int64 {
	n, sum := atomic.LoadInt64(&av.n), atomic.LoadInt64(&av.sum)
	if n == 0 {
		return 0
	}
	return int64(float64(sum) / float64(n))
}

func (av *averageInt64) variance() float64 {
	n, sumsq := atomic.LoadInt64(&av.n), atomic.LoadUint64(&av.sumsq)
	if n == 0 {
		return 0
	}
	n_f, mean_f := float64(n), float64(av.mean())
	sumsq_f := math.Float64frombits(sumsq)
	return (sumsq_f / n_f) - (mean_f * mean_f)
}

func (av *averageInt64) sd() float64 {
	n := atomic.LoadInt64(&av.n)
	if n == 0 {
		return 0
	}
	return math.Sqrt(av.variance())
}

func (av *averageInt64) clone() *averageInt64 {
	newav := (*av)
	return &newav
}
