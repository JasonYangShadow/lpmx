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
		err := ErrNew(ErrMarshal, "marshaling data error")
		return nil, &err
	}
}

func StructUnmarshal(data []byte) (interface{}, *Error) {
	var val interface{}
	err := msgpack.Unmarshal(data, &val)
	if err == nil {
		return val, nil
	} else {
		err := ErrNew(ErrMarshal, "unmarshaling data error")
		return nil, &err
	}
}
