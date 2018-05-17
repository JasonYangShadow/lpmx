package msgpack

import (
	. "github.com/jasonyangshadow/lpmx/error"
	"github.com/vmihailenco/msgpack"
)

func StructMarshal(data interface{}) ([]byte, *Error) {
	mp, err := msgpack.Marshal(data)
	if err == nil {
		return mp, nil
	} else {
		cerr := ErrNew(err, "marshaling data error")
		return nil, cerr
	}
}

func StructUnmarshal(data []byte, vtype interface{}) *Error {
	err := msgpack.Unmarshal(data, vtype)
	if err == nil {
		return nil
	}
	cerr := ErrNew(err, "unmarshaling to struct Container encounters error")
	return cerr
}
