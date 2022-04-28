package compose

import (
	"fmt"
	. "github.com/JasonYangShadow/lpmx/error"
	"github.com/agrison/go-commons-lang/stringUtils"
	"github.com/deckarep/golang-set"
	"strings"
)

const (
	VERSION = "1"
)

type TopLevel struct {
	Version string     `yaml:"version"`
	Apps    []AppLevel `yaml:"apps"`
}

type AppLevel struct {
	Name      string   `yaml:"name"`
	Image     string   `yaml:"image"`
	ImageType string   `yaml:"type"`
	Expose    []string `yaml:"expose"`
	Port      []string `yaml:"port"`
	DependsOn []string `yaml:"depends"`
}

func (topLevel *TopLevel) validate() *Error {
	if stringUtils.IsEmpty(topLevel.Version) || strings.TrimSpace(topLevel.Version) != VERSION {
		err := ErrNew(ErrNExist, "version in yaml file does not exist or is not correct")
		return err
	}

	nameset := mapset.NewSet()
	for _, element := range topLevel.Apps {
		if stringUtils.IsEmpty(element.Name) {
			err := ErrNew(ErrNExist, "name is a mandatory field")
			return err
		}

		if stringUtils.IsEmpty(element.Image) {
			err := ErrNew(ErrNExist, "image is a mandatory field")
			return err
		}

		if stringUtils.IsEmpty(element.ImageType) {
			err := ErrNew(ErrNExist, "image type is a mandatory field")
			return err
		}

		nameset.Add(element.Name)
	}

	for _, element := range topLevel.Apps {
		if len(element.DependsOn) > 0 {
			for _, dependItem := range element.DependsOn {
				if !nameset.Contains(dependItem) {
					err := ErrNew(ErrMismatch, fmt.Sprintf("%s is not defined in yaml file", dependItem))
					return err
				}
			}
		}
	}

	return nil
}
