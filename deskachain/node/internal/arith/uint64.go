package arith

import "errors"

var (
	ErrOverflow  = errors.New("uint64 overflow")
	ErrUnderflow = errors.New("uint64 underflow")
)

func Add(a, b uint64) (uint64, error) {
	if b > ^uint64(0)-a {
		return 0, ErrOverflow
	}
	return a + b, nil
}

func Sub(a, b uint64) (uint64, error) {
	if b > a {
		return 0, ErrUnderflow
	}
	return a - b, nil
}

func AddCap(a, b uint64) uint64 {
	value, err := Add(a, b)
	if err != nil {
		return ^uint64(0)
	}
	return value
}
