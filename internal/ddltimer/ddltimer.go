// Copyright 2023 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package ddltimer includes a [DeadlineTimer] that can be used to set deadlines and listen for time-out events. Here is
an example of how to use the DeadlineTimer:

	t := ddltimer.New()
	defer t.Stop()  // to prevent resource leaks
	t.SetDeadline(time.Now().Add(2 * time.Second))
	<-t.Timeout()   // will return after 2 seconds
	// you may also SetDeadline in other goroutines while waiting
*/
package ddltimer

import (
	"sync"
	"time"
)

// DeadlineTimer is a tool that allows you to set deadlines and listen for time-out events. It is more flexible than
// [time.After] because you can update the deadline; and it is more flexible than [time.Timer] because multiple
// subscribers can listen to the time-out channel.
//
// DeadlineTimer is safe for concurrent use by multiple goroutines.
//
// gvisor has a similar implementation: [gonet.deadlineTimer].
//
// [gonet.deadlineTimer]: https://github.com/google/gvisor/blob/release-20230605.0/pkg/tcpip/adapters/gonet/gonet.go#L130-L138
type DeadlineTimer struct {
	mu sync.Mutex

	ddl time.Time
	t   *time.Timer
	c   chan struct{}
}

// New creates a new instance of DeadlineTimer that can be used to SetDeadline() and listen for Timeout() events.
func New() *DeadlineTimer {
	return &DeadlineTimer{
		c: make(chan struct{}),
	}
}

// Timeout returns a readonly channel that will block until the specified amount of time set by SetDeadline() has
// passed. This channel can be safely subscribed to by multiple listeners.
//
// Timeout is similar to the [time.After] function.
func (d *DeadlineTimer) Timeout() <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.c
}

// SetDeadline changes the timer to expire after deadline t. When the timer expires, the Timeout() channel will be
// unblocked. A zero value means the timer will not time out.
//
// SetDeadline is like [time.Timer]'s Reset() function, but it doesn't have any restrictions.
func (d *DeadlineTimer) SetDeadline(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop the timer, and if d.t has already invoked the callback of AfterFunc, create a new channel.
	if d.t != nil && !d.t.Stop() {
		d.c = make(chan struct{})
	}

	// A second call to d.t.Stop() will return false, leading a never closed dangling channel.
	// See TestListenToMultipleTimeout() in ddltimer_test.go.
	d.t = nil

	// Handling the TestSetPastThenFuture() scenario in ddltimer_test.go:
	//   t := New()
	//   t.SetDeadline(yesterday)  // no d.t will be created, and we will close d.c
	//   t.SetDeadline(tomorrow)   // must handle the case of d.t==nil and d.c has been closed
	//   <-t.Timeout()             // should block until tomorrow
	select {
	case <-d.c:
		d.c = make(chan struct{})
	default:
	}

	d.ddl = t

	// A zero value means the timer will not time out.
	if t.IsZero() {
		return
	}

	timeout := time.Until(t)
	if timeout <= 0 {
		close(d.c)
		return
	}

	// Timer.Stop returns whether or not the AfterFunc has started, but does not indicate whether or not it has
	// completed. Make a copy of d.c to prevent close(ch) from racing with the next call of SetDeadline replacing d.c.
	ch := d.c
	d.t = time.AfterFunc(timeout, func() {
		close(ch)
	})
}

// Stop prevents the Timer from firing. It is equivalent to SetDeadline(time.Time{}).
func (d *DeadlineTimer) Stop() {
	d.SetDeadline(time.Time{})
}

// Deadline returns the current expiration time. If the timer will never expire, a zero value will be returned.
func (d *DeadlineTimer) Deadline() time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ddl
}
