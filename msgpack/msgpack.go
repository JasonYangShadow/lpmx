package msgpack

import (
	. "github.com/jasonyangshadow/lpmx/container"
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

func StructUnmarshal(data []byte, vtype interface{}) *Error {
	switch vtype.(type) {
	case *MemContainers:
		{
			err := msgpack.Unmarshal(data, vtype)
			if err == nil {
				return nil
			}
			cerr := ErrNew(err, "unmarshaling to struct MemContainers encounters error")
			return &cerr
		}
	case *Container:
		{
			err := msgpack.Unmarshal(data, vtype)
			if err == nil {
				return nil
			}
			cerr := ErrNew(err, "unmarshaling to struct Container encounters error")
			return &cerr
		}
	default:
		{
			cerr := ErrNew(ErrType, "unmarshaling to unknown struct")
			return &cerr
		}
	}
}
