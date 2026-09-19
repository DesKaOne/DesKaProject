package cpuminer

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"deskachain/internal/crypto"
	"deskachain/internal/types"
)

var ErrNonceExhausted = errors.New("nonce exhausted")

type Result struct {
	Block  types.Block
	Thread int
	Hashes uint64
}

func FirstNonces(threads int) []uint64 {
	if threads < 1 {
		threads = 1
	}
	nonces := make([]uint64, threads)
	for i := range nonces {
		nonces[i] = uint64(i)
	}
	return nonces
}

func Mine(ctx context.Context, block types.Block, threads int, totalHashes *atomic.Uint64) (Result, error) {
	if threads < 1 {
		threads = 1
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	target := strings.Repeat("0", int(block.Difficulty))
	results := make(chan Result, 1)
	errs := make(chan error, threads)
	var wg sync.WaitGroup
	for thread := 0; thread < threads; thread++ {
		thread := thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			for nonce := uint64(thread); ; nonce += uint64(threads) {
				select {
				case <-ctx.Done():
					return
				default:
				}
				hash := crypto.DoubleSHA256Hex(block.HeaderBytesWithNonce(nonce))
				if totalHashes != nil {
					totalHashes.Add(1)
				}
				if strings.HasPrefix(hash, target) {
					found := block
					found.Nonce = nonce
					found.Hash = hash
					select {
					case results <- Result{Block: found, Thread: thread}:
						cancel()
					default:
					}
					return
				}
				if nonce > math.MaxUint64-uint64(threads) {
					errs <- ErrNonceExhausted
					return
				}
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case result := <-results:
		<-done
		if totalHashes != nil {
			result.Hashes = totalHashes.Load()
		}
		return result, nil
	case <-done:
		select {
		case err := <-errs:
			return Result{}, err
		default:
			return Result{}, ctx.Err()
		}
	case <-ctx.Done():
		<-done
		return Result{}, ctx.Err()
	}
}
