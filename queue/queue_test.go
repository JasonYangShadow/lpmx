package queue

import (
	"testing"
)

func TestQueue1(t *testing.T) {
	q := InitQueue(DEFAULT_SIZE)
	q <- Queue{SYS}
	t.Log(<-q)
}
