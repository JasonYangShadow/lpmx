package compose

import (
	"fmt"
	. "github.com/JasonYangShadow/lpmx/error"
	"github.com/agrison/go-commons-lang/stringUtils"
	"github.com/deckarep/golang-set"
	"sort"
	"strings"
)

const (
	Version         = "1"
	TypeDocker      = "docker"
	TypeSingularity = "singularity"
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
	Share     []string `yaml:"share"`
	Inject    []string `yaml:"inject"`
	DependsOn []string `yaml:"depends"`
}

func (topLevel *TopLevel) Validate() ([]string, *map[string]AppLevel, *Error) {
	if stringUtils.IsEmpty(topLevel.Version) || strings.TrimSpace(topLevel.Version) != Version {
		err := ErrNew(ErrNExist, "version in yaml file does not exist or is not correct")
		return nil, nil, err
	}

	nameSet := mapset.NewSet()
	dependMap := make(map[string]int8)
	appsMap := make(map[string]AppLevel)
	for idx, element := range topLevel.Apps {
		if stringUtils.IsEmpty(element.Name) {
			err := ErrNew(ErrNExist, "name is a mandatory field")
			return nil, nil, err
		}

		if stringUtils.IsEmpty(element.Image) {
			err := ErrNew(ErrNExist, "image is a mandatory field")
			return nil, nil, err
		}

		if stringUtils.IsEmpty(element.ImageType) {
			err := ErrNew(ErrNExist, "image type is a mandatory field")
			return nil, nil, err
		}

		if element.ImageType != TypeDocker && element.ImageType != TypeSingularity {
			err := ErrNew(ErrMismatch, "image type should be either 'docker' or 'singularity'")
			return nil, nil, err
		}

		if len(element.Expose) > 0 {
			for _, expose := range element.Expose {
				if !stringUtils.Contains(expose, ":") {
					err := ErrNew(ErrNExist, "expose should contain ':'")
					return nil, nil, err
				}
			}
		}

		if len(element.Port) > 0 {
			for _, port := range element.Port {
				if !stringUtils.Contains(port, ":") {
					err := ErrNew(ErrNExist, "port should contain ':'")
					return nil, nil, err
				}
			}
		}

		if len(element.Share) > 0 {
			for _, share := range element.Share {
				if !stringUtils.Contains(share, ":") {
					err := ErrNew(ErrNExist, "share should contain ':'")
					return nil, nil, err
				}
			}
		}

		if len(element.Inject) > 0 {
			for _, inject := range element.Inject {
				if !stringUtils.Contains(inject, ":") {
					err := ErrNew(ErrNExist, "inject should contain ':'")
					return nil, nil, err
				}
			}
		}

		if nameSet.Contains(element.Name) {
			err := ErrNew(ErrExist, fmt.Sprintf("%s is already defined in yaml, should not have duplicated name", element.Name))
			return nil, nil, err
		}

		nameSet.Add(element.Name)

		appsMap[element.Name] = topLevel.Apps[idx]
	}

	for _, element := range topLevel.Apps {
		if len(element.DependsOn) > 0 {
			for _, dependItem := range element.DependsOn {
				if !nameSet.Contains(dependItem) {
					err := ErrNew(ErrMismatch, fmt.Sprintf("%s is not defined in yaml file", dependItem))
					return nil, nil, err
				}

				if v, ok := dependMap[dependItem]; ok {
					dependMap[dependItem] = v + 1
				} else {
					dependMap[dependItem] = 1
				}
			}
		}
	}

	if len(dependMap) > 0 {
		keys := make([]string, 0, len(dependMap))
		for key := range dependMap {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return dependMap[keys[i]] > dependMap[keys[j]]
		})
		return keys, &appsMap, nil
	}

	return nil, &appsMap, nil
}
