package breaker

import "errors"

// Breaker tracks the state of the circuit breaker. All properties are
// maintained internally to allow the circuit breaker to be used in a
// concurrent environment.
type Breaker struct {
	failCount    int
	successCount int
	state        State
	shouldTrip   stateFunc
}

// A StateFunc defines a function that can be used to determine a state
// change.
type stateFunc func() bool

// State represents the state of the circuit breaker
type State int

// Constants defining the state of the circuit breaker
const (
	StateOpen State = iota
	StateClosed
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// NewBreaker returns a default instance of the circuit breaker. Unless
// configured otherwise, the breaker will open after 5 failed transactions.
// By default the broker will need to be manually reset.
func NewBreaker() *Breaker {

	b := Breaker{}
	b.state = StateClosed
	b.TripAfter(5)
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
}

// success increments the successCount
func (b *Breaker) success() {
	b.successCount++
}

// Protect wraps a function that returns an error with the circuit
// breaker. If an error is returned, the breaker increments the
// failure counter. If a success is returned, the breaker increments
// the success counter.
//
// If the breaker is open, an error is returned indicating the current
// state of the breaker.
func (b *Breaker) Protect(f func() error) error {

	if b.CurrentState() == StateOpen {
		return errors.New("breaker open")
	}

	err := f()
	if err != nil {
		b.fail()

		if b.shouldTrip() == true {
			b.state = StateOpen
		}

		return err
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
