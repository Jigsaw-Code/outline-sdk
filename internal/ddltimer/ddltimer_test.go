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

package ddltimer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var zeroDeadline = time.Time{}

func TestNew(t *testing.T) {
	d := New()
	assert.Equal(t, d.Deadline(), zeroDeadline)
	select {
	case <-d.Timeout():
		assert.Fail(t, "d.Timeout() should never be fired")
	case <-time.After(1 * time.Second):
		assert.Equal(t, d.Deadline(), zeroDeadline)
	}
}

func TestSetDeadline(t *testing.T) {
	d := New()
	start := time.Now()
	d.SetDeadline(start.Add(200 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(200*time.Millisecond))

	<-d.Timeout()
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 200*time.Millisecond)
	assert.Less(t, duration, 300*time.Millisecond)
	assert.Equal(t, d.Deadline(), start.Add(200*time.Millisecond))
}

func TestSetDeadlineInGoRoutine(t *testing.T) {
	d := New()
	start := time.Now()
	go func() {
		time.Sleep(200 * time.Millisecond) // make sure SetDeadline is called after d.Timeout()
		assert.Equal(t, d.Deadline(), zeroDeadline)
		d.SetDeadline(start.Add(400 * time.Millisecond))
		assert.Equal(t, d.Deadline(), start.Add(400*time.Millisecond))
	}()

	<-d.Timeout()
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 400*time.Millisecond)
	assert.Less(t, duration, 500*time.Millisecond)
	assert.Equal(t, d.Deadline(), start.Add(400*time.Millisecond))
}

func TestStop(t *testing.T) {
	d := New()
	start := time.Now()
	d.SetDeadline(start.Add(200 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(200*time.Millisecond))
	d.Stop()
	assert.Equal(t, d.Deadline(), zeroDeadline)
	select {
	case <-d.Timeout():
		assert.Fail(t, "d.Timeout() should never be fired")
	case <-time.After(1 * time.Second):
		assert.Equal(t, d.Deadline(), zeroDeadline)
	}
}

func TestStopInGoRoutine(t *testing.T) {
	d := New()
	start := time.Now()
	d.SetDeadline(start.Add(500 * time.Millisecond))
	go func() {
		time.Sleep(300 * time.Millisecond) // make sure Stop is called after d.Timeout()
		assert.Equal(t, d.Deadline(), start.Add(500*time.Millisecond))
		d.Stop()
		assert.Equal(t, d.Deadline(), zeroDeadline)
	}()

	select {
	case <-d.Timeout():
		assert.Fail(t, "d.Timeout() should never be fired")
	case <-time.After(1 * time.Second):
		assert.Equal(t, d.Deadline(), zeroDeadline)
	}
}

func TestSetPastThenFuture(t *testing.T) {
	d := New()
	start := time.Now()
	d.SetDeadline(start.Add(-500 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(-500*time.Millisecond))
	d.SetDeadline(start.Add(500 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(500*time.Millisecond))

	<-d.Timeout()
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 500*time.Millisecond)
	assert.Less(t, duration, 600*time.Millisecond)
	assert.Equal(t, d.Deadline(), start.Add(500*time.Millisecond))
}

func TestSetFutureThenPast(t *testing.T) {
	d := New()
	start := time.Now()
	d.SetDeadline(start.Add(500 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(500*time.Millisecond))
	d.SetDeadline(start.Add(-100 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(-100*time.Millisecond))

	<-d.Timeout()
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 0*time.Second)
	assert.Less(t, duration, 100*time.Millisecond)
	assert.Equal(t, d.Deadline(), start.Add(-100*time.Millisecond))
}

func TestSetDeadlineSequence(t *testing.T) {
	d := New()
	start := time.Now()
	d.SetDeadline(start.Add(100 * time.Millisecond))
	ch1 := d.Timeout()
	<-ch1
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
	assert.Less(t, duration, 150*time.Millisecond)
	assert.Equal(t, d.Deadline(), start.Add(100*time.Millisecond))

	start2 := time.Now()
	d.SetDeadline(start2.Add(100 * time.Millisecond))
	ch2 := d.Timeout()
	assert.NotEqual(t, ch1, ch2)
	<-ch1
	<-ch2
	duration = time.Since(start)
	assert.GreaterOrEqual(t, duration, 200*time.Millisecond)
	assert.Less(t, duration, 250*time.Millisecond)
	assert.Equal(t, d.Deadline(), start2.Add(100*time.Millisecond))
}

func TestListenToMultipleTimeout(t *testing.T) {
	d := New()
	start := time.Now()
	ch0 := d.Timeout()

	d.SetDeadline(start.Add(100 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(100*time.Millisecond))
	ch1 := d.Timeout()
	assert.Equal(t, ch0, ch1)

	d.Stop()
	assert.Equal(t, d.Deadline(), zeroDeadline)
	ch2 := d.Timeout()
	assert.Equal(t, ch0, ch2)
	assert.Equal(t, ch1, ch2)

	d.Stop()
	assert.Equal(t, d.Deadline(), zeroDeadline)
	ch3 := d.Timeout()
	assert.Equal(t, ch0, ch3)
	assert.Equal(t, ch1, ch3)
	assert.Equal(t, ch2, ch3)

	d.SetDeadline(start.Add(300 * time.Millisecond))
	assert.Equal(t, d.Deadline(), start.Add(300*time.Millisecond))
	ch4 := d.Timeout()
	assert.Equal(t, ch3, ch4)
	assert.Equal(t, ch0, ch4)
	assert.Equal(t, ch1, ch4)
	assert.Equal(t, ch2, ch4)

	// All timeout channels must be fired
	<-ch0
	<-ch1
	<-ch2
	<-ch3
	<-ch4
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 300*time.Millisecond)
	assert.Less(t, duration, 350*time.Millisecond)
	assert.Equal(t, d.Deadline(), start.Add(300*time.Millisecond))
}
