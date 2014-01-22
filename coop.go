// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// coop contains some of the most commonly used concurrent
// flows. This package is mostly around for reference. even though
// you can use it as a library, I'd suggest you to implement your own
// control mechanisms.
package coop

import (
	"sync"
	"time"
)

// Runs fn at the specified time.
func At(t time.Time, fn func()) (done <-chan bool) {
	return After(t.Sub(time.Now()), fn)
}

// Runs until time in every dur.
func Until(t time.Time, dur time.Duration, fn func()) (done <-chan bool) {
	ch := make(chan bool)
	go untilRecv(ch, t, dur, fn)
	return ch
}

func untilRecv(ch chan bool, t time.Time, dur time.Duration, fn func()) {
	if t.Sub(time.Now()) > 0 {
		time.AfterFunc(dur, func() {
			fn()
			untilRecv(ch, t, dur, fn)
		})
		return
	}
	ch <- true
}

// Runs fn after duration. Similar to time.AfterFunc
func After(duration time.Duration, fn func()) (done <-chan bool) {
	ch := make(chan bool)
	time.AfterFunc(duration, func() {
		fn()
		ch <- true
	})
	return ch
}

// Runs fn in every specified duration.
func Every(dur time.Duration, fn func()) {
	time.AfterFunc(dur, func() {
		fn()
		Every(dur, fn)
	})
}

// Runs fn and times out if it runs longer than the provided
// duration. It will send false to the returning
// channel if timeout occurs.
// TODO: cancel if timeout occurs
func Timeout(duration time.Duration, fn func()) (done <-chan bool) {
	ch := make(chan bool)
	go func() {
		<-time.After(duration)
		ch <- false
	}()
	go func() {
		fn()
		ch <- true
	}()
	return ch
}

// Starts a job and returns a channel for cancellation signal.
// Once a message is sent to the channel, stops the fn.
func Killable(fn func()) (kill chan<- bool, done <-chan bool) {
	ch := make(chan bool)
	dch := make(chan bool)
	go func() {
		<-dch
		ch <- false
	}()
	go func() {
		fn()
		ch <- true
	}()
	return dch, ch
}

// Starts to run the given list of fns concurrently.
func All(fns ...func()) (done <-chan bool) {
	var wg sync.WaitGroup
	wg.Add(len(fns))

	ch := make(chan bool)
	for _, fn := range fns {
		go func(f func()) {
			f()
			wg.Done()
		}(fn)
	}
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}

// Starts to run the given list of fns concurrently,
// at most n fns at a time.
func AllWithThrottle(throttle int, fns ...func()) (done <-chan bool) {
	ch := make(chan bool)
	go func() {
		for {
			num := throttle
			if throttle > len(fns) {
				num = len(fns)
			}
			next := fns[:num]
			fns = fns[num:]
			<-All(next...)
			if len(fns) == 0 {
				ch <- true
				break
			}
		}
	}()
	return ch
}

// Run the same function with n copies.
func Replicate(n int, fn func()) (done <-chan bool) {
	var wg sync.WaitGroup
	wg.Add(n)
	ch := make(chan bool)
	for i := 0; i < n; i++ {
		go func() {
			fn()
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}
