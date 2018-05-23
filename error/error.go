package error

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
)

var (
	ErrNil       = errors.New("Nil error")
	ErrType      = errors.New("Type error")
	ErrFull      = errors.New("Space full error")
	ErrNExist    = errors.New("Not exist error")
	ErrExist     = errors.New("Exist error")
	ErrMismatch  = errors.New("Type mismatch error")
	ErrFileStat  = errors.New("file stat error")
	ErrUnknown   = errors.New("Unknown error")
	ErrDirMake   = errors.New("Error when making a folder")
	ErrMarshal   = errors.New("Error while marshaling or unmarshaling data")
	ErrFileIO    = errors.New("Error while reading or writing files")
	ErrCmd       = errors.New("Error while running cmd")
	ErrStatus    = errors.New("Status not satisfied")
	ErrRPCServer = errors.New("RPC server created error")
)

type Error struct {
	Err error
	Msg *list.List
}

func ErrNew(err error, msg string) *Error {
	cerr := new(Error)
	cerr.Err = err
	cerr.Msg = list.New()
	cerr.Msg.PushBack(msg)
	return cerr
}

func (e *Error) Error() string {
	var buffer bytes.Buffer
	for m := e.Msg.Front(); m != nil; m = m.Next() {
		buffer.WriteString(m.Value.(string))
		buffer.WriteString("\n")
	}
	return fmt.Sprintf("[ErrType: %s], [ErrMsg: \n%s]", e.Err.Error(), buffer.String())
}

func (e *Error) AddMsg(str string) {
	e.Msg.PushBack(str)
}
