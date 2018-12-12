package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/log"
	"github.com/phayes/permbits"
	"github.com/sirupsen/logrus"
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
	TYPE_OTHER
)

func FileExist(file string) bool {
	ftype, err := FileType(file)
	if err == nil && (ftype == TYPE_REGULAR || ftype == TYPE_OTHER) {
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
		return TYPE_OTHER, nil
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

func GetFilePermission(file interface{}) (uint32, *Error) {
	switch file.(type) {
	case string:
		permissions, err := permbits.Stat(file.(string))
		if err != nil {
			cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.stat %s error: %s", file, err.Error()))
			return 1, cerr
		}
		return uint32(permissions), nil
	case os.FileInfo:
		fileMode := file.(os.FileInfo).Mode()
		permissions := permbits.FileMode(fileMode)
		return uint32(permissions), nil
	default:
		cerr := ErrNew(ErrMismatch, "file type is not in (string, os.FileInfo)")
		return 1, cerr
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

func GuessPath(base string, in string, file bool) (string, *Error) {
	if strings.HasPrefix(in, "$") {
		return strings.Replace(in, "$", "", -1), nil
	}
	if strings.HasPrefix(in, "^") {
		in = strings.Replace(in, "^", "", -1)
		file = true
	}
	if strings.TrimSpace(in) == "all" {
		return in, nil
	}
	var str string
	if filepath.IsAbs(in) {
		str = in
	} else {
		str = filepath.Join(base, in)
	}
	if (file && FileExist(str)) || (!file && FolderExist(str)) {
		return str, nil
	}
	cerr := ErrNew(ErrNil, fmt.Sprintf("%s doesn't exist both in abs path and relative path", str))
	return "", cerr
}

func GuessPathContainer(base string, layers []string, in string, file bool) (string, *Error) {
	if strings.HasPrefix(in, "$") {
		return strings.Replace(in, "$", "", -1), nil
	}
	if strings.HasPrefix(in, "^") {
		in = strings.Replace(in, "^", "", -1)
		file = true
	}
	if strings.TrimSpace(in) == "all" {
		return in, nil
	}
	if filepath.IsAbs(in) {
		if (file && FileExist(in)) || (!file && FolderExist(in)) {
			return in, nil
		}
	} else {
		for _, layer := range layers {
			tpath := fmt.Sprintf("%s/%s/%s", base, layer, in)
			LOGGER.WithFields(logrus.Fields{
				"tpath": tpath,
			}).Debug("guess path container debug")
			if (file && FileExist(tpath)) || (!file && FolderExist(tpath)) {
				return tpath, nil
			}
		}
	}
	cerr := ErrNew(ErrNil, fmt.Sprintf("%s doesn't exist both in abs path and relative path", in))
	return "", cerr
}

//get all existed paths rather than only one
func GuessPathsContainer(base string, layers []string, in string, file bool) ([]string, *Error) {
	var ret []string
	if strings.HasPrefix(in, "$") {
		ret = append(ret, strings.Replace(in, "$", "", -1))
		return ret, nil
	}
	//file marked with ^
	if strings.HasPrefix(in, "^") {
		in = strings.Replace(in, "^", "", -1)
		file = true
	}
	if strings.TrimSpace(in) == "all" {
		ret = append(ret, in)
		return ret, nil
	}
	if filepath.IsAbs(in) {
		if (file && FileExist(in)) || (!file && FolderExist(in)) {
			ret = append(ret, in)
		}
	} else {
		for _, layer := range layers {
			tpath := fmt.Sprintf("%s/%s/%s", base, layer, in)
			LOGGER.WithFields(logrus.Fields{
				"tpath": tpath,
			}).Debug("guess paths container debug")
			if (file && FileExist(tpath)) || (!file && FolderExist(tpath)) {
				ret = append(ret, tpath)
			}
		}
	}
	if len(ret) > 0 {
		return ret, nil
	}
	cerr := ErrNew(ErrNil, fmt.Sprintf("%s doesn't exist both in abs path and relative path", in))
	return ret, cerr
}

func AddConPath(base string, in string) string {
	if strings.HasPrefix(in, "$") {
		return strings.Replace(in, "$", "", -1)
	}
	return filepath.Join(base, in)
}

func Tar(src string, writers ...io.Writer) *Error {
	// ensure the src actually exists before trying to tar it
	if !FolderExist(src) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s folder not exist", src))
		return cerr
	}

	mw := io.MultiWriter(writers...)
	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err := filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}
		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		f.Close()
		return nil
	})
	if err != nil {
		cerr := ErrNew(err, "tar file error")
		return cerr
	}
	return nil
}

func Untar(target string, folder string) *Error {
	if !FileExist(target) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("file %s does not exist", target))
		return cerr
	}
	r, err := os.Open(target)
	if err != nil {
		cerr := ErrNew(ErrFileIO, fmt.Sprintf("open file %s failure", target))
		return cerr
	}
	defer r.Close()
	gzr, err := gzip.NewReader(r)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("gzip open file %s failure", target))
		return cerr
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			cerr := ErrNew(err, "reading tar header errors")
			return cerr

		case header == nil:
			continue
		}

		if !strings.HasSuffix(folder, "/") {
			folder = folder + "/"
		}
		target := filepath.Join(folder, header.Name)

		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					cerr := ErrNew(err, "untar making dir error")
					return cerr
				}
			}

		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				cerr := ErrNew(err, fmt.Sprintf("untar create file %s error", target))
				return cerr
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				cerr := ErrNew(err, "untar copying file content error")
				return cerr
			}

			f.Close()

		case tar.TypeSymlink:
			//won't link to absolute path
			real_path := header.Linkname
			if strings.HasPrefix(real_path, "/") {
				real_path = fmt.Sprintf("%s%s", folder, strings.TrimPrefix(header.Linkname, "/"))
			}
			os.Symlink(real_path, target)

		case tar.TypeLink:
			//won't link to absolute path
			real_path := header.Linkname
			if strings.HasPrefix(real_path, "/") {
				real_path = fmt.Sprintf("%s%s", folder, strings.TrimPrefix(header.Linkname, "/"))
			}
			os.Link(real_path, target)
		}
	}
}

func ReverseStrArray(input []string) []string {
	for i := 0; i < len(input)/2; i++ {
		j := len(input) - i - 1
		input[i], input[j] = input[j], input[i]
	}
	return input
}
