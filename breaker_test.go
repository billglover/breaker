package breaker

import (
	"errors"
	"log"
	"testing"
	"time"
)

func errorFunc() error {
	return errors.New("protected service failure")
}

func successFunc() error {
	return nil
}

func TestStates(t *testing.T) {
	if StateClosed.String() != "closed" {
		t.Fatalf("unexpected state description: want %s, got %s", "closed", StateClosed.String())
	}

	if StateOpen.String() != "open" {
		t.Fatalf("unexpected state description: want %s, got %s", "open", StateOpen.String())
	}

	if StatePartial.String() != "partial" {
		t.Fatalf("unexpected state description: want %s, got %s", "partial", StatePartial.String())
	}

	if State(30).String() != "unknown" {
		t.Fatalf("unexpected state description: want %s, got %s", "unknown", State(30).String())
	}
}

func TestNewBreaker(t *testing.T) {
	cb := NewBreaker()

	if cb.FailCount() != 0 {
		t.Fatalf("unexpected initial fail count: want %d, got %d", 0, cb.FailCount())
	}

	if cb.SuccessCount() != 0 {
		t.Fatalf("unexpected initial success count: want %d, got %d", 0, cb.SuccessCount())
	}

	if cb.CurrentState() != StateClosed {
		t.Fatalf("unexpected initial state: want %v, got %v", StateClosed, cb.CurrentState())
	}
}

func ExampleNewBreaker() {
	cb := NewBreaker()

	log.Printf("fail count: %d\n", cb.FailCount())
}

func TestFailCount(t *testing.T) {
	cb := NewBreaker()

	cb.fail()

	if cb.FailCount() != 1 {
		t.Fatalf("unexpected fail count: want %d, got %d", 1, cb.FailCount())
	}

	cb.fail()
	cb.fail()

	if cb.FailCount() != 3 {
		t.Fatalf("unexpected fail count: want %d, got %d", 3, cb.FailCount())
	}

}

func TestSuccessCount(t *testing.T) {
	cb := NewBreaker()

	cb.success()

	if cb.SuccessCount() != 1 {
		t.Fatalf("unexpected success count: want %d, got %d", 1, cb.SuccessCount())
	}

	cb.success()
	cb.success()

	if cb.SuccessCount() != 3 {
		t.Fatalf("unexpected success count: want %d, got %d", 3, cb.SuccessCount())
	}
}

func TestReset(t *testing.T) {
	cb := NewBreaker()
	cb.success()
	cb.fail()
	cb.fail()

	if cb.SuccessCount() != 1 {
		t.Fatalf("unexpected success count: want %d, got %d", 1, cb.SuccessCount())
	}

	if cb.FailCount() != 2 {
		t.Fatalf("unexpected fail count: want %d, got %d", 2, cb.FailCount())
	}

	cb.Reset()

	if cb.FailCount() != 0 {
		t.Fatalf("unexpected final fail count: want %d, got %d", 0, cb.FailCount())
	}

	if cb.SuccessCount() != 0 {
		t.Fatalf("unexpected final success count: want %d, got %d", 0, cb.SuccessCount())
	}
}

func TestProtectError(t *testing.T) {
	cb := NewBreaker()

	err := cb.Protect(func() error {
		return errorFunc()
	})

	if err == nil {
		t.Fatalf("unexpected response: no error returned")
	}

	if cb.FailCount() != 1 {
		t.Fatalf("unexpected fail count: want %d, got %d", 1, cb.FailCount())
	}

	if cb.SuccessCount() != 0 {
		t.Fatalf("unexpected success count: want %d, got %d", 0, cb.SuccessCount())
	}

}

func TestProtectSuccess(t *testing.T) {
	cb := NewBreaker()

	err := cb.Protect(func() error {
		return successFunc()
	})

	if err != nil {
		t.Fatalf("unexpected response: %v", err)
	}

	if cb.FailCount() != 0 {
		t.Fatalf("unexpected fail count: want %d, got %d", 0, cb.FailCount())
	}

	if cb.SuccessCount() != 1 {
		t.Fatalf("unexpected success count: want %d, got %d", 1, cb.SuccessCount())
	}
}

func TestTripAfter(t *testing.T) {
	cb := NewBreaker().TripAfter(6)

	outcomes := []bool{false, true, false, false, false, false, false}

	for _, o := range outcomes {
		cb.Protect(func() error {
			if o {
				return successFunc()
			}
			return errorFunc()
		})
	}

	if cb.FailCount() != 6 {
		t.Fatalf("unexpected fail count: want %d, got %d", 6, cb.FailCount())
	}

	if cb.SuccessCount() != 1 {
		t.Fatalf("unexpected success count: want %d, got %d", 2, cb.SuccessCount())
	}

	if cb.CurrentState() != StateOpen {
		t.Fatalf("unexpected final state: want %v, got %v", StateOpen, cb.CurrentState())
	}

	err := cb.Protect(func() error {
		return errorFunc()
	})
	if err == nil {
		t.Fatalf("unexpected response: no error returned")
	}
}

func TestResetAfterSuccess(t *testing.T) {
	cb := NewBreaker().TripAfter(3)
	outcomes := []bool{true, false, false, false}

	for _, o := range outcomes {
		cb.Protect(func() error {
			if o {
				return successFunc()
			}
			return errorFunc()
		})
	}

	// confirm that we enter the open state after a series of failed transactions
	if cb.CurrentState() != StateOpen {
		t.Fatalf("unexpected state: want %v, got %v", StateOpen, cb.CurrentState())
	}

	err := cb.Protect(func() error {
		return successFunc()
	})
	if err == nil {
		t.Fatalf("unexpected response: no error returned")
	}

	// wait 50ms and confirm that the breaker has entered the partially open state
	time.Sleep(50 * time.Millisecond)

	err = cb.Protect(func() error {
		return successFunc()
	})

	if err != nil {
		t.Fatalf("unexpected response: %v", err)
	}
	if cb.CurrentState() != StateClosed {
		t.Fatalf("unexpected final state: want %v, got %v", StateClosed, cb.CurrentState())
	}

	// confirm the counters end up where we would expect
	if cb.FailCount() != 0 {
		t.Fatalf("unexpected final fail count: want %d, got %d", 0, cb.FailCount())
	}

	if cb.SuccessCount() != 1 {
		t.Fatalf("unexpected final success count: want %d, got %d", 1, cb.SuccessCount())
	}
}

func TestResetAfterFail(t *testing.T) {
	cb := NewBreaker().TripAfter(3)
	outcomes := []bool{true, false, false, false}

	for _, o := range outcomes {
		cb.Protect(func() error {
			if o {
				return successFunc()
			}
			return errorFunc()
		})
	}

	// confirm that we enter the open state after a series of failed transactions
	if cb.CurrentState() != StateOpen {
		t.Fatalf("unexpected state: want %v, got %v", StateOpen, cb.CurrentState())
	}

	err := cb.Protect(func() error {
		return successFunc()
	})
	if err == nil {
		t.Fatalf("unexpected response: no error returned")
	}

	// wait 50ms and confirm that the breaker has reset
	time.Sleep(50 * time.Millisecond)

	err = cb.Protect(func() error {
		return errorFunc()
	})
	if err == nil {
		t.Fatalf("unexpected response: no error returned")
	}

	if cb.CurrentState() != StateOpen {
		t.Fatalf("unexpected final state: want %v, got %v", StateOpen, cb.CurrentState())
	}

	if cb.FailCount() != 1 {
		t.Fatalf("unexpected final fail count: want %d, got %d", 1, cb.FailCount())
	}

	if cb.SuccessCount() != 0 {
		t.Fatalf("unexpected final success count: want %d, got %d", 0, cb.SuccessCount())
	}
}

func TestSubscribe(t *testing.T) {

	cb := NewBreaker()
	c1 := cb.Subscribe()

	// create a second subscriber but don't drain notifications
	cb.Subscribe()

	cb.trip()
	s1 := <-c1

	if s1 != StateOpen {
		t.Fatalf("unexpected notification received: want %s, got %s", StateOpen, s1)
	}

	cb.partial()
	s1 = <-c1
	if s1 != StatePartial {
		t.Fatalf("unexpected notification received: want %s, got %s", StatePartial, s1)
	}

	cb.Reset()
	s1 = <-c1
	if s1 != StateClosed {
		t.Fatalf("unexpected notification received: want %s, got %s", StateClosed, s1)
	}
}

func ExampleBreaker_TripAfter() {
	cb := NewBreaker().TripAfter(5)

	err := cb.Protect(func() error {
		// make the function call you are trying to protect
		// and return an error on failure
		return nil
	})

	if err != nil {
		log.Println(err)
	}
}

func ExampleBreaker_ResetAfter() {
	cb := NewBreaker().TripAfter(5).ResetAfter(50 * time.Millisecond)

	err := cb.Protect(func() error {
		// make the function call you are trying to protect
		// and return an error on failure
		return nil
	})

	if err != nil {
		log.Println(err)
	}
}

func ExampleBreaker_Protect() {
	cb := NewBreaker()

	err := cb.Protect(func() error {
		return errors.New("protected call failed")
	})

	if err != nil {
		log.Println(err)
	}
}

func ExampleBreaker_Subscribe() {

	// create a circuit breaker
	cb := NewBreaker().TripAfter(1).ResetAfter(10 * time.Millisecond)

	// subscribe to notifications
	n := cb.Subscribe()
	go func(c <-chan State) {
		for s := range c {
			log.Println(s)
		}
	}(n)

	// simulate a call to a protected function

	cb.Protect(func() error {
		return errors.New("protected call failed")
	})

	// notify: open

}
