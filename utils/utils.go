package utils

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"github.com/phayes/permbits"
	"os"
)

const (
	PERM_WRITE = iota
	PERM_READ
	PERM_EXE
	PERM_WRITE_READ
	PERM_WRITE_EXE
	PERM_READ_EXE

	TYPE_REGULAR = iota
	TYPE_DIR
	TYPE_SYMLINK
	TYPE_PIPE
)

func FileExist(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func FileType(file string) (int8, *Error) {
	fi, err := os.Stat(file)
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.stat %s error: %s", file, err.Error()))
		return -1, &cerr
	}
	switch mode := fi.Mode(); {
	case mode.IsRegular():
		return TYPE_REGULAR, nil
	case mode.IsDir():
		return TYPE_DIR, nil
	case mode&os.ModeSymlink != 0:
		return TYPE_SYMLINK, nil
	case mode&os.ModeNamedPipe != 0:
		return TYPE_PIPE, nil
	default:
		cerr := ErrNew(ErrNExist, fmt.Sprintf("file mode is not recognized %s ", file))
		return -1, &cerr
	}
}

func FilePermission(file interface{}, permType int8) (bool, *Error) {
	switch file.(type) {
	case string:
		permissions, err := permbits.Stat(file.(string))
		if err != nil {
			cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.stat %s error: %s", file, err.Error()))
			return false, &cerr
		}
		switch permType {
		case PERM_WRITE:
			return permissions.UserWrite(), nil
		case PERM_READ:
			return permissions.UserRead(), nil
		case PERM_EXE:
			return permissions.UserExecute(), nil
		case PERM_WRITE_READ:
			return permissions.UserWrite() && permissions.UserRead(), nil
		case PERM_READ_EXE:
			return permissions.UserRead() && permissions.UserExecute(), nil
		case PERM_WRITE_EXE:
			return permissions.UserWrite() && permissions.UserExecute(), nil
		default:
			cerr := ErrNew(ErrNExist, "permTyoe doesn't exist")
			return false, &cerr
		}
	case os.FileInfo:
		fileMode := file.(os.FileInfo).Mode()
		permissions := permbits.FileMode(fileMode)
		switch permType {
		case PERM_WRITE:
			return permissions.UserWrite(), nil
		case PERM_READ:
			return permissions.UserRead(), nil
		case PERM_EXE:
			return permissions.UserExecute(), nil
		case PERM_WRITE_READ:
			return permissions.UserWrite() && permissions.UserRead(), nil
		case PERM_READ_EXE:
			return permissions.UserRead() && permissions.UserExecute(), nil
		case PERM_WRITE_EXE:
			return permissions.UserWrite() && permissions.UserExecute(), nil
		default:
			cerr := ErrNew(ErrNExist, "permTyoe doesn't exist")
			return false, &cerr
		}

	default:
		cerr := ErrNew(ErrMismatch, "file type is not in (string, os.FileInfo)")
		return false, &cerr
	}

}

func MakeDir(dir string) (bool, *Error) {
	err := os.MkdirAll(dir, 0777)
	if err == nil {
		return true, nil
	}
	cerr := ErrNew(ErrDirMake, fmt.Sprintf("creating %s folder error", dir))
	return false, &cerr
}
