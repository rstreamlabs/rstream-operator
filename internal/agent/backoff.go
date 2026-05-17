// See LICENSE file in the project root for license information.

package agent

import "time"

type backoff struct {
	initial time.Duration
	max     time.Duration
	next    time.Duration
}

func newBackoff(initial, max time.Duration) *backoff {
	return &backoff{initial: initial, max: max, next: initial}
}

func (b *backoff) Next() time.Duration {
	if b == nil {
		return time.Second
	}
	current := b.next
	b.next *= 2
	if b.next > b.max {
		b.next = b.max
	}
	return current
}

func (b *backoff) Reset() {
	if b == nil {
		return
	}
	b.next = b.initial
}
