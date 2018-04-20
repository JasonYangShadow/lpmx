package queue

const (
	CONFIG_CREATE = iota
	CONFIG_MODIFY
	CONFIG_REMOVE
	SYS

	DEFAULT_SIZE = 100
)

type Queue struct {
	Msg int
}

var (
	MSG = []string{"config add msg", "config modify msg", "config delete msg", "sys"}
)

func InitQueue(size int) chan Queue {
	q := make(chan Queue, size)
	return q
}
