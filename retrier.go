package graphql

// Retrier indicates whether or not to retry and attempt another query.
type Retrier interface {
	// ShouldRetry returns true if another attempt should run,
	// given 'err' from the previous attempt and the total attempt count (starts at 1).
	//
	// Consider the 'errors' package to unwrap the error. e.g. errors.As(), errors.Is()
	ShouldRetry(err error, attempts uint) bool
}

var _ Retrier = CountRetrier{}

// CountRetrier is a Retrier that stops after a number of attempts.
type CountRetrier struct {
	// maxAttempts is the maximum number of attempts allowed before retries should stop.
	// A value of 0 has undefined behavior.
	maxAttempts uint
}

// NewCountRetrier returns a CountRetrier with the given maximum number of retries
// beyond the first attempt.
func NewCountRetrier(maxRetries uint) CountRetrier {
	return CountRetrier{
		maxAttempts: 1 + maxRetries,
	}
}

func (c CountRetrier) ShouldRetry(err error, attempts uint) bool {
	return attempts < c.maxAttempts
}
