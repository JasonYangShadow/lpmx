package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/JasonYangShadow/lpmx/error"
	. "github.com/JasonYangShadow/lpmx/log"
	. "github.com/JasonYangShadow/lpmx/paeudo"
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
	PERM_READ_WRITE_EXE

	TYPE_REGULAR = iota
	TYPE_DIR
	TYPE_SYMLINK
	TYPE_PIPE
	TYPE_SOCKET
	TYPE_OTHER
)

var (
	memcached_checklist = []string{"memcached", "libevent"}
	time_sleep          = 2
)

func FileExist(file string) bool {
	ftype, err := FileType(file)
	if err == nil && (ftype == TYPE_REGULAR || ftype == TYPE_SOCKET || ftype == TYPE_OTHER || ftype == TYPE_SYMLINK) {
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
	case mode&os.ModeSocket != 0:
		return TYPE_SOCKET, nil
	default:
		return TYPE_OTHER, nil
	}
}

func ChangeFilePermssion(file string, perm uint32) (bool, *Error) {
	err := permbits.Chmod(file, permbits.PermissionBits(perm))
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.chmod %s error: %s", file, err.Error()))
		return false, cerr
	}
	return true, nil
}

func CheckFilePermission(file string, perm uint32, force bool) (bool, *Error) {
	permissions, err := permbits.Stat(file)
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.chmod %s error: %s", file, err.Error()))
		return false, cerr
	}

	if permissions != permbits.PermissionBits(perm) {
		if force {
			return ChangeFilePermssion(file, perm)
		}
		return false, nil
	}
	return true, nil
}

func WalkandCheckFilePermission(folder string, checklist []string, perm uint32, force bool) (bool, *Error) {
	if !FolderExist(folder) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s does not exist", folder))
		return false, cerr
	}
	bool_checklist := make([]bool, len(checklist))
	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		for idx, item := range checklist {
			if strings.HasPrefix(info.Name(), item) {
				bool_checklist[idx] = true
				if ret, err := CheckFilePermission(path, perm, force); !ret {
					if err != nil {
						return err.Err
					}
				}
				return nil
			}
		}
		return nil
	})

	for idx, item := range bool_checklist {
		if !item {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("necessary file %s does not exist", checklist[idx]))
			return false, cerr
		}
	}
	return true, nil

}

//only check user permission
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
		case PERM_READ_WRITE_EXE:
			return permissions.UserRead() && permissions.UserWrite() && permissions.UserExecute(), nil
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
		case PERM_READ_WRITE_EXE:
			return permissions.UserRead() && permissions.UserWrite() && permissions.UserExecute(), nil
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

func GetFileSize(file interface{}) (int64, *Error) {
	switch file.(type) {
	case string:
		fi, err := os.Stat(file.(string))
		if err != nil {
			cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.stat %s error: %s", file, err.Error()))
			return 1, cerr
		}
		return fi.Size(), nil
	case os.FileInfo:
		return file.(os.FileInfo).Size(), nil
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

func Rename(old_path string, new_path string) *Error {
	if t, terr := FileType(old_path); terr == nil {
		if t == TYPE_DIR {
			parent_dir := filepath.Dir(new_path)
			if !FolderExist(parent_dir) {
				err := os.MkdirAll(parent_dir, 0777)
				if err != nil {
					cerr := ErrNew(err, fmt.Sprintf("could not mkdir %s", parent_dir))
					return cerr
				}
			}
		}
		if t == TYPE_REGULAR {
			parent_dir := filepath.Dir(new_path)
			if !FolderExist(parent_dir) {
				err := os.MkdirAll(parent_dir, 0777)
				if err != nil {
					cerr := ErrNew(err, fmt.Sprintf("could not mkdir %s", parent_dir))
					return cerr
				}
			}
		}
	}
	err := os.Rename(old_path, new_path)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not rename from: %s, to: %s", old_path, new_path))
		return cerr
	}
	return nil
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

func TarFiles(filelist []string, target_folder string, target_name string) *Error {
	target_path := fmt.Sprintf("%s/%s.tar.gz", target_folder, target_name)
	file, ferr := os.Create(target_path)
	if ferr != nil {
		cerr := ErrNew(ferr, fmt.Sprintf("%s creating error", target_path))
		return cerr
	}
	defer file.Close()

	mw := io.Writer(file)
	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for _, fitem := range filelist {
		fi, err := os.Stat(fitem)
		if err != nil {
			cerr := ErrNew(ErrFileStat, fmt.Sprintf("os.stat %s error: %s", fitem, err.Error()))
			return cerr
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			cerr := ErrNew(err, "could not get file header")
			return cerr
		}
		//modify header's name
		header.Name = filepath.Base(fitem)
		if err := tw.WriteHeader(header); err != nil {
			cerr := ErrNew(err, "could not write tar file header")
			return cerr
		}

		if !fi.Mode().IsRegular() {
			cerr := ErrNew(ErrType, fmt.Sprintf("%s is not regular file, only supports regular file compression", fitem))
			return cerr
		}

		f, err := os.Open(fitem)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not open file %s", fitem))
			return cerr
		}
		if _, err := io.Copy(tw, f); err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not read file %s", fitem))
			return cerr
		}
		f.Close()
	}

	return nil
}

//this tar function eliminate symlink
func TarLayer(src_folder string, target_folder string, target_name string, layers []string) *Error {
	if !FolderExist(src_folder) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s folder not exist", src_folder))
		return cerr
	}

	target_path := fmt.Sprintf("%s/%s.tar.gz", target_folder, target_name)
	file, ferr := os.Create(target_path)
	if ferr != nil {
		cerr := ErrNew(ferr, fmt.Sprintf("%s creating error", target_path))
		return cerr
	}
	defer file.Close()

	mw := io.Writer(file)
	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err := filepath.Walk(src_folder, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file == src_folder {
			return nil
		}

		//if symlink
		mode := fi.Mode()
		if mode&os.ModeSymlink != 0 {
			link, err := os.Readlink(file)
			if err != nil {
				return err
			}
			for _, layer_path := range layers {
				if strings.HasPrefix(link, layer_path) {
					//let's create another symlink instead of this real one
					//step 1 : rename real one
					//step 2 : create new fake one with the same name
					//step 3 : remove new fake one and restore old.one
					err := os.Rename(file, fmt.Sprintf("%s.tar.old", file))
					if err != nil {
						return err
					}
					relative_path, err := filepath.Rel(link, layer_path)
					if err != nil {
						return err
					}
					if relative_path == "." {
						relative_path = "/"
					} else {
						relative_path = fmt.Sprintf("/%s", relative_path)
					}

					err = os.Symlink(relative_path, file)
					new_fi, err := os.Lstat(file)
					if err != nil {
						return err
					}
					//new header
					header, err := tar.FileInfoHeader(new_fi, fi.Name())
					if err != nil {
						return err
					}
					header.Name = strings.TrimPrefix(file, src_folder)
					if err := tw.WriteHeader(header); err != nil {
						return err
					}

					//remove fake file and restore original one
					err = os.Remove(file)
					if err != nil {
						return err
					}
					err = os.Rename(fmt.Sprintf("%s.tar.old", file), file)
					if err != nil {
						return err
					}
					//done
					return nil
				}
			}
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}
		//modify header's name
		header.Name = strings.TrimPrefix(file, src_folder)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			//do not need copy
			return nil
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
			//if linkname is absolute path should be linked to the same layer
			if strings.HasPrefix(header.Linkname, "/") {
				os.Symlink(fmt.Sprintf("%s%s", strings.TrimSuffix(folder, "/"), header.Linkname), target)
			} else {
				os.Symlink(header.Linkname, target)
			}

			//we should avoid of creating hard link
		case tar.TypeLink:
			if strings.HasPrefix(header.Linkname, "/") {
				os.Symlink(fmt.Sprintf("%s%s", strings.TrimSuffix(folder, "/"), header.Linkname), target)
			} else {
				os.Symlink(header.Linkname, target)
			}
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

func GetCurrDir() (string, *Error) {
	arg_path, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	searchPaths := []string{arg_path, "."}
	searchPaths = append(searchPaths, strings.Split(os.Getenv("PATH"), ":")...)
	for _, path := range searchPaths {
		p := fmt.Sprintf("%s/lpmx", path)
		if FileExist(p) {
			return path, nil
		}
	}
	cerr := ErrNew(ErrNExist, fmt.Sprintf("can't locate lpmx among PATH, caller directory and current directroy"))
	return "", cerr
}

func AddVartoFile(env string, file string) *Error {
	perm, err := GetFilePermission(file)
	if err != nil {
		return err
	}
	f, er := os.OpenFile(file, os.O_APPEND|os.O_RDWR, os.FileMode(perm))
	if er != nil {
		cerr := ErrNew(er, fmt.Sprintf("can't open file %s", file))
		return cerr
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, fmt.Sprintf("export %s", env)) {
			return nil
		}
	}

	_, er = f.WriteString(fmt.Sprintf("export %s\n", env))
	if er != nil {
		cerr := ErrNew(er, fmt.Sprintf("can't write file %s", file))
		return cerr
	}
	return nil
}

func GetHostOSInfo() (string, string, *Error) {
	output, err := Command("lsb_release", "-a")
	if err != nil {
		return "", "", err
	}

	distributor := ""
	release := ""
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Distributor ID:") {
			distributor = strings.TrimSpace(strings.TrimPrefix(line, "Distributor ID:"))
		}

		if strings.HasPrefix(line, "Release:") {
			release = strings.TrimSpace(strings.TrimPrefix(line, "Release:"))
		}
	}

	return distributor, release, nil
}

func GetProcessIdByName(name string) (bool, string, *Error) {
	cmd_context := fmt.Sprintf("ps -ef|grep %s|grep -v grep|awk '{print $2}'", name)
	out, err := CommandBash(cmd_context)
	if err != nil {
		return false, "", err
	}

	if out == "" {
		return false, out, nil
	} else {
		return true, out, nil
	}
}

func CheckProcessByPid(pid string) (bool, *Error) {
	cmd_context := fmt.Sprintf("ps -p %s --no-headers", pid)
	out, err := CommandBash(cmd_context)
	if err != nil {
		return false, err
	}

	if out == "" {
		return false, nil
	} else {
		return true, nil
	}
}

func KillProcessByPid(pid string) *Error {
	cmd_context := fmt.Sprintf("kill -9 %s", pid)
	_, err := CommandBash(cmd_context)
	if err != nil {
		return err
	}
	return nil
}

func CheckCompleteness(folder string, checklist []string) *Error {
	if !FolderExist(folder) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s folder does not exist", folder))
		return cerr
	}
	bool_checklist := make([]bool, len(checklist))
	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		for idx, item := range checklist {
			if strings.HasPrefix(info.Name(), item) {
				bool_checklist[idx] = true
			}
		}
		return nil
	})

	for idx, item := range bool_checklist {
		if !item {
			cerr := ErrNew(ErrNExist, fmt.Sprintf("necessary file %s does not exist", checklist[idx]))
			return cerr
		}
	}
	return nil
}

func CheckAndStartMemcache() *Error {
	if ok, _, _ := GetProcessIdByName("memcached"); !ok {
		currdir, _ := GetCurrDir()
		currdir = fmt.Sprintf("%s/.lpmxsys", currdir)

		cerr := CheckCompleteness(currdir, memcached_checklist)
		if cerr == nil {
			_, cerr := CommandBash(fmt.Sprintf("LD_PRELOAD=%s/libevent.so %s/memcached -s %s/.memcached.pid -a 600 -d", currdir, currdir, currdir))
			if cerr != nil {
				cerr.AddMsg(fmt.Sprintf("can not start memcached process from %s", currdir))
				return cerr
			}
			memcached_pid_path := fmt.Sprintf("%s/.memcached.pid", currdir)
			//delay several seconds to detect .memcached.pid exists
			time.Sleep(time.Duration(time_sleep) * time.Second)
			if !FileExist(memcached_pid_path) {
				cerr = ErrNew(ErrNExist, fmt.Sprintf("could not find pid file: %s, memcached starts failure", memcached_pid_path))
				return cerr
			}
		}
		return cerr
	}
	return nil
}

func Sha256file(file string) (string, *Error) {
	f, err := os.Open(file)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not open file %s", file))
		return "", cerr
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not calculate sha256 value of %s", file))
		return "", cerr
	}

	value := fmt.Sprintf("%x", h.Sum(nil))
	return value, nil
}

func Sha256str(str string) (string, *Error) {
	h := sha256.New()
	if _, err := io.WriteString(h, str); err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not calculate sha256 value of %s", str))
		return "", cerr
	}

	value := fmt.Sprintf("%x", h.Sum(nil))
	return value, nil
}
