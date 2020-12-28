package filecache

import (
	"fmt"
	"os"
	"strings"

	. "github.com/JasonYangShadow/lpmx/error"
	. "github.com/JasonYangShadow/lpmx/utils"
)

type FileCacheInst struct {
	FilePath string
}

func FInitServer(path string) (*FileCacheInst, *Error) {
	var fc FileCacheInst
	fc.FilePath = path
	return &fc, nil
}

func (fc *FileCacheInst) FGetStrValue(key string) (string, *Error) {
	if !FileExist(fc.FilePath) {
		file, ferr := os.Create(fc.FilePath)
		if ferr != nil {
			cerr := ErrNew(ferr, fmt.Sprintf("could not create file: %s", fc.FilePath))
			return "", cerr
		}
		file.Close()
		return "", nil
	}
	content, err := ReadFromFile(fc.FilePath)
	if err != nil {
		return "", err
	}
	text := string(content)
	for _, item := range strings.Split(text, ":") {
		kv := strings.Split(item, "=")
		if len(kv) > 0 && kv[0] == key {
			return kv[1], nil
		}
	}
	return "", nil
}

func (fc *FileCacheInst) FSetValue(key string, value string) *Error {
	texts := []string{}
	if !FileExist(fc.FilePath) {
		file, ferr := os.Create(fc.FilePath)
		if ferr != nil {
			cerr := ErrNew(ferr, fmt.Sprintf("could not create file: %s", fc.FilePath))
			return cerr
		}
		file.Close()
	} else {
		content, err := ReadFromFile(fc.FilePath)
		if err != nil {
			return err
		}
		texts = strings.Split(string(content), ":")
		for idx, item := range texts {
			kv := strings.Split(item, "=")
			if len(kv) > 0 && kv[0] == key {
				//we find the duplicated key
				texts[idx] = fmt.Sprintf("%s=%s", kv[0], value)
				//write back
				err = WriteToFile([]byte(strings.Join(texts, ":")), fc.FilePath)
				if err != nil {
					return err
				}
				return nil
			}
		}

	}

	//append to the end
	if len(texts) == 0 {
		texts = []string{fmt.Sprintf("%s=%s", key, value)}
	} else {
		texts = append(texts, fmt.Sprintf("%s=%s", key, value))
	}
	err := WriteToFile([]byte(strings.Join(texts, ":")), fc.FilePath)
	if err != nil {
		return err
	}
	return nil
}

func (fc *FileCacheInst) FDeleteByKey(key string) *Error {
	if !FileExist(fc.FilePath) {
		file, ferr := os.Create(fc.FilePath)
		if ferr != nil {
			cerr := ErrNew(ferr, fmt.Sprintf("could not create file: %s", fc.FilePath))
			return cerr
		}
		file.Close()
		return nil
	}
	content, err := ReadFromFile(fc.FilePath)
	if err != nil {
		return err
	}
	texts := strings.Split(string(content), ":")
	for idx, item := range texts {
		kv := strings.Split(item, "=")
		if len(kv) > 0 && kv[0] == key {
			texts[len(texts)-1], texts[idx] = texts[idx], texts[len(texts)-1]
			texts = texts[:len(texts)-1]
			//write back
			err = WriteToFile([]byte(strings.Join(texts, ":")), fc.FilePath)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}
