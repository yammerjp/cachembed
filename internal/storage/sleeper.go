package storage

import "time"

type Sleeper interface {
	Sleep(d time.Duration)
}

type RealSleeper struct{}

func (s RealSleeper) Sleep(d time.Duration) {
	time.Sleep(d)
}
