// Package errgroup provides a Group that is capable of collecting errors
// as it waits for a collection of goroutines to finish.
package errgroup

import (
	"bytes"
	"sync"
)

// MultiError allows returning a group of errors as one error.
type MultiError []error

// Error returns a concatenated string of all contained errors.
func (m MultiError) Error() string {
	l := len(m)
	if l == 0 {
		panic("MultiError with no errors")
	}
	if l == 1 {
		panic("MultiError with only 1 error")
	}
	var b bytes.Buffer
	b.WriteString("multiple errors: ")
	for i, e := range m {
		b.WriteString(e.Error())
		if i != l-1 {
			b.WriteString(" | ")
		}
	}
	return b.String()
}

// NewMultiError returns nil if all input errors passed in are nil. Otherwise,
// it coalesces all input errors into a single error instance. Useful for
// code like this:
//
//	func doThisAndThat() error {
//	  err1 := tryThis()
//	  err2 := tryThat()
//	  return errgroup.NewMultiError(err1, err2)
//	}
func NewMultiError(errs ...error) error {
	var multiErr MultiError
	for _, err := range errs {
		if err != nil {
			multiErr = append(multiErr, err)
		}
	}

	if len(multiErr) == 1 {
		return multiErr[0]
	} else if len(multiErr) > 1 {
		return multiErr
	}
	return nil
}

// Group is similar to a sync.WaitGroup, but allows for collecting errors.
// The collected errors are never reset, so unlike a sync.WaitGroup, this Group
// can only be used _once_. That is, you may only call Wait on it once.
type Group struct {
	wg     sync.WaitGroup
	mu     sync.Mutex
	errors MultiError
}

// Add adds delta, which may be negative. See sync.WaitGroup.Add documentation
// for details.
func (g *Group) Add(delta int) {
	g.wg.Add(delta)
}

// Done decrements the Group counter.
func (g *Group) Done() {
	g.wg.Done()
}

// Error adds an error to return in Wait. `nil` errors are ignored.
func (g *Group) Error(e error) {
	if e == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.errors = append(g.errors, e)
}

// Finish adds an error if not nil, and calls Done.
func (g *Group) Finish(e error) {
	g.Error(e)
	g.Done()
}

// Wait blocks until the Group counter is zero. If no errors were recorded, it
// returns nil. If one error was recorded, it returns it as is. If more than
// one error was recorded it returns a MultiError which is a slice of errors.
func (g *Group) Wait() error {
	g.wg.Wait()
	g.mu.Lock()
	defer g.mu.Unlock()
	errors := g.errors
	l := len(errors)
	if l == 0 {
		return nil
	}
	if l == 1 {
		return errors[0]
	}
	return errors
}
