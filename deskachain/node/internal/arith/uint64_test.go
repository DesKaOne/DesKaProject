package arith

import (
	"errors"
	"math"
	"testing"
)

func TestAdd(t *testing.T) {
	got, err := Add(1, 2)
	if err != nil || got != 3 {
		t.Fatalf("Add(1,2) = %d, %v", got, err)
	}
}

func TestAddOverflow(t *testing.T) {
	if _, err := Add(math.MaxUint64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("expected overflow, got %v", err)
	}
	if _, err := Add(math.MaxUint64-1, 1); err != nil {
		t.Fatalf("boundary add rejected: %v", err)
	}
	if got := AddCap(math.MaxUint64, 1); got != math.MaxUint64 {
		t.Fatalf("AddCap overflow = %d", got)
	}
}

func TestSub(t *testing.T) {
	got, err := Sub(5, 3)
	if err != nil || got != 2 {
		t.Fatalf("Sub(5,3) = %d, %v", got, err)
	}
}

func TestSubUnderflow(t *testing.T) {
	if _, err := Sub(0, 1); !errors.Is(err, ErrUnderflow) {
		t.Fatalf("expected underflow, got %v", err)
	}
}
