package main

import "math"
import "fmt"
import "sync/atomic"

// Average maintains the average and variance of a stream
// of numbers in a space-efficient manner.
type Average struct {
	count uint64
	sum   uint64
	sumsq uint64
}

// Add a sample to counting average.
func (av *Average) Add(sample uint64) {
	atomic.AddUint64(&av.count, 1)
	atomic.AddUint64(&av.sum, sample)
	atomic.AddUint64(&av.sumsq, sample*sample)
}

// GetCount return the number of samples counted so far.
func (av *Average) Count() uint64 {
	return av.count
}

// Mean return the sum of all samples by number of samples so far.
func (av *Average) Mean() float64 {
	return float64(av.sum) / float64(av.count)
}

// GetTotal return the sum of all samples so far.
func (av *Average) Sum() uint64 {
	return av.sum
}

// Variance return the variance of all samples so far.
func (av *Average) Variance() float64 {
	a := av.Mean()
	return float64(av.sumsq)/float64(av.count) - a*a
}

// GetStdDev return the standard-deviation of all samples so far.
func (av *Average) Sd() float64 {
	return math.Sqrt(av.Variance())
}

func (av *Average) String() string {
	fmsg := "n:%v mean:%v var:%v sd:%v"
	return fmt.Sprintf(fmsg, av.Count(), av.Mean(), av.Variance(), av.Sd())
}
