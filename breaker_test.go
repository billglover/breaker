package breaker

import (
	"errors"
	"testing"
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
		t.Fatalf("unexpected fail count: want %d, got %d", 5, cb.FailCount())
	}

	if cb.SuccessCount() != 1 {
		t.Fatalf("unexpected success count: want %d, got %d", 2, cb.SuccessCount())
	}

	if cb.CurrentState() != StateOpen {
		t.Fatalf("unexpected final state: want %v, got %v", StateOpen, cb.CurrentState())
	}

	err := cb.Protect(func() error {
		return successFunc()
	})
	if err == nil {
		t.Fatalf("unexpected response: no error returned")
	}
}
