package error

import (
	"errors"
	"fmt"
)

var (
	ErrNil      = errors.New("Nil error")
	ErrType     = errors.New("Type error")
	ErrFull     = errors.New("Space full error")
	ErrNExist   = errors.New("Not exist error")
	ErrExist    = errors.New("Exist error")
	ErrMismatch = errors.New("Type mismatch error")
	ErrFileStat = errors.New("file stat error")
)

type Error struct {
	err error
	msg string
}

func ErrNew(err error, msg string) Error {
	cerr := Error{err, msg}
	return cerr
}

func (e *Error) Error() string {
	return fmt.Sprintf("{\"ErrType\":\"%s\", \"ErrMsg\":\"%s\"}", e.err.Error(), e.msg)
}
