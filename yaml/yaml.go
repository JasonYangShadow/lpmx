package yaml

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"github.com/spf13/viper"
	"path/filepath"
	"strings"
)

func SetLocalValue(file string, key string, value interface{}) *Error {
	viper.SetConfigFile("yaml")
	viper.AddConfigPath(".")
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in local dir", file))
		return cerr
	}
	viper.Set(key, value)
	return nil
}

func GetLocalValue(file string, key string) (interface{}, *Error) {
	viper.SetConfigFile("yaml")
	viper.AddConfigPath(".")
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in local dir", file))
		return nil, cerr
	}
	return viper.Get(key), nil
}

func GetLocalStrValue(file string, key string) (string, *Error) {
	viper.SetConfigFile("yaml")
	viper.AddConfigPath(".")
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in local dir", file))
		return "", cerr
	}
	return viper.GetString(key), nil
}

func SetValue(file string, config []string, key string, value interface{}) *Error {
	viper.SetConfigFile("yaml")
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return cerr
	}
	viper.Set(key, value)
	return nil
}

func GetValue(file string, config []string, key string) (interface{}, *Error) {
	viper.SetConfigFile("yaml")
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return "", cerr
	}
	return viper.Get(key), nil
}

func GetStrValue(file string, config []string, key string) (string, *Error) {
	viper.SetConfigFile("yaml")
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return "", cerr
	}
	return viper.GetString(key), nil
}

func GetMap(file string, config []string) (map[string]interface{}, *Error) {
	viper.SetConfigFile("yaml")
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return nil, cerr
	}
	return viper.AllSettings(), nil
}

func MultiGetMap(file string, config []string) (*viper.Viper, map[string]interface{}, *Error) {
	v := viper.New()
	v.SetConfigType("yaml")
	for i := range config {
		v.AddConfigPath(config[i])
	}
	v.SetConfigName(file)
	err := v.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return v, nil, cerr
	}
	return v, v.AllSettings(), nil
}

func LoadConfig(config string) (*viper.Viper, map[string]interface{}, *Error) {
	v := viper.New()
	v.SetConfigType("yaml")
	dir, file := filepath.Split(config)
	file = strings.TrimSuffix(file, ".yml")
	v.SetConfigName(file)
	v.AddConfigPath(dir)
	err := v.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return v, nil, cerr
	}
	return v, v.AllSettings(), nil

}
