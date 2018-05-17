package utils

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"github.com/phayes/permbits"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
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
	ftype, err := FileType(file)
	if err == nil && ftype == TYPE_REGULAR {
		return true
	}
	return false
}

func FolderExist(folder string) bool {
	ftype, err := FileType(folder)
	if err == nil && ftype == TYPE_DIR {
		return true
	}
	return false
}

func FileType(file string) (int8, *Error) {
	fi, err := os.Stat(file)
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.stat %s error: %s", file, err.Error()))
		return -1, cerr
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
		return -1, cerr
	}
}

func FilePermission(file interface{}, permType int8) (bool, *Error) {
	switch file.(type) {
	case string:
		permissions, err := permbits.Stat(file.(string))
		if err != nil {
			cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.stat %s error: %s", file, err.Error()))
			return false, cerr
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
			return false, cerr
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
			return false, cerr
		}

	default:
		cerr := ErrNew(ErrMismatch, "file type is not in (string, os.FileInfo)")
		return false, cerr
	}

}

func MakeDir(dir string) (bool, *Error) {
	err := os.MkdirAll(dir, 0777)
	if err == nil {
		return true, nil
	}
	cerr := ErrNew(err, fmt.Sprintf("creating %s folder error", dir))
	return false, cerr
}

func RemoveAll(dir string) (bool, *Error) {
	err := os.RemoveAll(dir)
	if err == nil {
		return true, nil
	}
	cerr := ErrNew(err, fmt.Sprintf("removing %s folder error", dir))
	return false, cerr
}

func RemoveFile(path string) (bool, *Error) {
	err := os.Remove(path)
	if err == nil {
		return true, nil
	}
	cerr := ErrNew(err, fmt.Sprintf("removing %s file error", path))
	return false, cerr
}

func ReadFromFile(dir string) ([]byte, *Error) {
	data, err := ioutil.ReadFile(dir)
	if err == nil {
		return data, nil
	} else {
		err := ErrNew(ErrFileIO, fmt.Sprintf("reading file %s error", dir))
		return nil, err
	}
}

func WriteToFile(data []byte, dir string) *Error {
	err := ioutil.WriteFile(dir, data, 0644)
	if err == nil {
		return nil
	} else {
		err := ErrNew(ErrFileIO, fmt.Sprintf("writing file %s error", dir))
		return err
	}
}

func CopyFile(src string, dst string) (bool, *Error) {
	if !FileExist(src) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("source file %s doesn't exist", src))
		return false, cerr
	}
	ft, err := FileType(src)
	if err != nil {
		cerr := ErrNew(ErrType, fmt.Sprintf("checking source file %s type encounters error", src))
		return false, cerr
	}
	if ft != TYPE_REGULAR {
		cerr := ErrNew(ErrType, fmt.Sprintf("source file %s is not regular type file", src))
		return false, cerr
	}
	if FileExist(dst) {
		cerr := ErrNew(ErrExist, fmt.Sprintf("target file %s exist, can't override", src))
		return false, cerr
	}
	in, ierr := os.Open(src)
	if ierr != nil {
		cerr := ErrNew(ierr, fmt.Sprintf("can't open file %s", src))
		return false, cerr
	}
	defer in.Close()
	out, oerr := os.Create(dst)
	if oerr != nil {
		cerr := ErrNew(oerr, fmt.Sprintf("can't open file %s", dst))
		return false, cerr
	} else {
		defer out.Close()
	}

	if _, yerr := io.Copy(out, in); err != nil {
		cerr := ErrNew(yerr, fmt.Sprintf("copy file encounters error src: %s, dst: %s", src, dst))
		return false, cerr
	}
	si, _ := os.Stat(src)
	merr := os.Chmod(dst, si.Mode())
	if merr != nil {
		cerr := ErrNew(merr, fmt.Sprintf("can't change the permission of file %s", dst))
		return false, cerr
	}

	serr := out.Sync()
	if err != nil {
		cerr := ErrNew(serr, fmt.Sprintf("can't sync file %s", dst))
		return false, cerr
	}

	return true, nil
}

func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func RandomPort(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
