package yaml

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"github.com/spf13/viper"
)

func SetLocalValue(file string, key string, value interface{}) *Error {
	viper.AddConfigPath(".")
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in local dir", file))
		return &cerr
	}
	viper.Set(key, value)
	return nil
}

func GetLocalValue(file string, key string) (interface{}, *Error) {
	viper.AddConfigPath(".")
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in local dir", file))
		return nil, &cerr
	}
	return viper.Get(key), nil
}

func GetLocalStrValue(file string, key string) (string, *Error) {
	viper.AddConfigPath(".")
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in local dir", file))
		return "", &cerr
	}
	return viper.GetString(key), nil
}

func SetValue(file string, config []string, key string, value interface{}) *Error {
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return &cerr
	}
	viper.Set(key, value)
	return nil
}

func GetValue(file string, config []string, key string) (interface{}, *Error) {
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return "", &cerr
	}
	return viper.Get(key), nil
}

func GetStrValue(file string, config []string, key string) (string, *Error) {
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return "", &cerr
	}
	return viper.GetString(key), nil
}

func GetMap(file string, config []string) (map[string]interface{}, *Error) {
	for i := range config {
		viper.AddConfigPath(config[i])
	}
	viper.SetConfigName(file)
	err := viper.ReadInConfig()
	if err != nil {
		cerr := ErrNew(ErrNil, fmt.Sprintf("can't open file %s in dirs", file))
		return nil, &cerr
	}
	return viper.AllSettings(), nil
}
