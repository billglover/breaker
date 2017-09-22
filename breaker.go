/*
Package breaker provides a simple implementation of the circuite breaker
pattern. A typical use would be to protect a call to an external system.
By protecting calls to external systems with a circuit breaker, you are
able to respond quickly in the event of system failure, avoiding the
need to wait for a timeout.

   cb := NewBreaker().TripAfter(5).ResetAfter(500)

Further information on the circuit breaker pattern can be found in the
Microsoft Azure Architecture Patterns documentation.

https://docs.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker
*/
package breaker

import (
	"errors"
	"time"
)

// Breaker represents a circuit breaker. In normal use, an instance of
// the circuit breaker should be used to protect a single external
// system. Protecting multiple systems with a single instance of a
// circuit breaker is not recommended.
type Breaker struct {
	failCount    int
	successCount int
	lastFail     time.Time
	state        State
	shouldTrip   stateFunc
	shouldReset  stateFunc
	subscribers  []chan State
}

// A StateFunc defines a function that can be used to determine a state
// change.
type stateFunc func() bool

// State represents the state of the circuit breaker
type State int

// Circuit breaker states
const (
	StateOpen State = iota
	StateClosed
	StatePartial
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StatePartial:
		return "partial"
	default:
		return "unknown"
	}
}

// NewBreaker returns an instance of a circuit breaker using the default
// configuration.
//
// By default the circuit breaker will trip after 5 failed transactions,
// enter the partially open state after 50ms. Once in the partially open
// state it will reset if the next call is successful or trip if it fails.
func NewBreaker() *Breaker {

	b := Breaker{}
	b.state = StateClosed
	b.TripAfter(5)
	b.ResetAfter(50 * time.Millisecond)
	return &b
}

// FailCount returns the current count of failed transactions.
func (b *Breaker) FailCount() int {
	return b.failCount
}

// SuccessCount returns the current count of successful transactions.
func (b *Breaker) SuccessCount() int {
	return b.successCount
}

// CurrentState returns the current state of the circuit breaker.
func (b *Breaker) CurrentState() State {
	return b.state
}

// fail increments the failCount
func (b *Breaker) fail() {
	b.failCount++
	b.lastFail = time.Now()
}

// success increments the successCount
func (b *Breaker) success() {
	b.successCount++
}

// Reset returns the fail and success counters to zero
func (b *Breaker) Reset() {
	b.state = StateClosed
	b.failCount = 0
	b.successCount = 0
	b.notify(StateClosed)
}

// partial returns the fail and success counters to zero
func (b *Breaker) partial() {
	b.state = StatePartial
	b.failCount = 0
	b.successCount = 0
	b.notify(StatePartial)
}

// trip opens the breaker
func (b *Breaker) trip() {
	b.state = StateOpen
	b.notify(StateOpen)
}

// Protect wraps a function that returns an error with the circuit
// breaker. If an error is returned, the breaker increments the
// failure counter. If a success is returned, the breaker increments
// the success counter.
//
// If the breaker is open, an error is returned indicating the current
// state of the breaker.
func (b *Breaker) Protect(f func() error) error {

	// if the breaker is open and we are ready to reset then enter the
	// partially open state
	if b.CurrentState() == StateOpen {
		if b.shouldReset() == false {
			return errors.New("breaker open")
		}
		b.partial()
	}

	// pass through the next request and handle the response based on
	// the current state of the breaker
	err := f()
	if err != nil {
		b.fail()

		if b.CurrentState() == StatePartial {
			b.trip()
		}

		if b.shouldTrip() == true {
			b.trip()
		}

		return err
	}

	// if we are in the partial state then reset the breaker
	if b.CurrentState() == StatePartial {
		b.Reset()
	}

	b.success()
	return nil
}

// TripAfter configures the breaker to trip after n failed transactions.
// Note that these failed transactions do not need to occur consecutively.
func (b *Breaker) TripAfter(n int) *Breaker {
	b.shouldTrip = func() bool {
		return b.FailCount() >= n
	}
	return b
}

// ResetAfter configures the breaker to reset after a period of time since
// the last failure.
func (b *Breaker) ResetAfter(t time.Duration) *Breaker {
	b.shouldReset = func() bool {
		resetTime := b.lastFail.Add(t)
		if time.Now().After(resetTime) {
			return true
		}
		return false
	}
	return b
}

// Subscribe returns a channel on which consumers can receive notifications
// on state change.
func (b *Breaker) Subscribe() chan State {
	c := make(chan State, 1)
	b.subscribers = append(b.subscribers, c)
	return c
}

func (b *Breaker) notify(state State) {
	for _, s := range b.subscribers {

	out:
		// Drain the channels before sending a notification.
		// This prevents blocking if notifications aren't
		// consumed.
		for {
			select {
			case <-s:
			default:
				break out
			}
		}
		s <- state
	}
}
