package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ManifestItem struct {
	download    string
	checksum    string
	tags        []string
	description string
	dependency  map[string]ManifestItem
}

func (m ManifestItem) Download(outputDir, lib, name string) error {
	content, err := getBody(m.download)
	if err != nil {
		return err
	}

	if fmt.Sprintf("%x", sha256.Sum256(content)) != m.checksum {
		return errors.New("failed checksum when downloading '" + lib + "/" + name + "' component")
	}

	for depName, dep := range m.dependency {
		if err = dep.Download(outputDir, lib, depName); err != nil {
			return err
		}
	}

	return os.WriteFile(filepath.Join(outputDir, lib, lib+"-"+name+".wed.html"), content, 0666)
}

type Collection map[string]map[string]ManifestItem

func (manifests Collection) Search(npattern, tpattern string) (results Collection, err error) {
	nsearch, err := regexp.Compile(npattern)
	if err != nil {
		return
	}

	tsearch, err := regexp.Compile(tpattern)
	if err != nil {
		return
	}

	results = make(map[string]map[string]ManifestItem)

	for lib, manifest := range manifests {
		for name, item := range manifest {
			var valid = false

			if tpattern != "" {
				for _, tag := range item.tags {
					if tsearch.MatchString(tag) {
						valid = true
						break
					}
				}
				if !valid {
					continue
				}
			}

			for _, tag := range item.tags {
				if nsearch.MatchString(tag) {
					valid = true
					break
				}
			}
			if valid = valid || nsearch.MatchString(name); !valid {
				continue
			}

			res, ok := results[lib]
			if !ok {
				res = make(map[string]ManifestItem, 1)
				results[lib] = res
			}
			res[name] = item
		}
	}

	return
}

func (manifests Collection) Find(search string) (lib, name string, selected ManifestItem, err error) {
	var results []string

	for lib, manifest := range manifests {
		for n, _ := range manifest {
			if n == search {
				name = n
				results = append(results, lib)
			}
		}
	}
	switch len(results) {
	case 0:
		err = errors.New("cannot find any component named '" + name + "' on trusted libraries")
	case 1:
		selected = manifests[results[0]][name]
	default:
		err = errors.New(strconv.Itoa(len(results)) + " matching components on different libraries: '" + strings.Join(results, ", ") + "'")
	}

	return
}

func LoadCollection() (manifests Collection, err error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	configDir = filepath.Join(configDir, "wednesday", "trusted")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return
	}

	var content []byte
	manifests = make(map[string]map[string]ManifestItem)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		lib := entry.Name()
		content, err = os.ReadFile(filepath.Join(configDir, lib))
		if err != nil {
			return
		}

		lib = cutExt(lib)
		manifest := make(map[string]ManifestItem)
		if err = json.Unmarshal(content, &manifest); err != nil {
			return
		}
		manifests[lib] = manifest
	}

	return
}

func doLibSearch(s Settings) (err error) {
	coll, err := LoadCollection()
	if err != nil {
		return
	}

	results, err := coll.Search(s.arg, s.tags)
	if err != nil {
		return
	}

	for lib, manifest := range results {
		for name, item := range manifest {
			fmt.Println(">", lib+"/"+name, item.tags, "\n", item.description)
		}
	}

	return
}

func doLibTrust(s Settings) (err error) {
	var content []byte
	if strings.HasPrefix(s.arg, "http") {
		content, err = getBody(s.arg)
	} else if s.arg != "" {
		content, err = os.ReadFile(s.arg)
	} else {
		return errors.New("missing manifest reference")
	}

	if err != nil {
		return
	}

	if err = json.Unmarshal(content, &map[string]ManifestItem{}); err != nil {
		return
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	return os.WriteFile(
		filepath.Join(configDir, "wednesday", "trusted", s.name+".json"),
		content,
		0666,
	)
}

func doLibDistrust(lib string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	return os.Remove(filepath.Join(configDir, "wednesday", "trusted", lib+".json"))
}

func cutExt(s string) string {
	return s[:len(s)-len(filepath.Ext(s))]
}

func getBody(link string) (content []byte, err error) {
	res, err := http.Get(link)
	if err != nil {
		return
	}

	if 200 <= res.StatusCode && res.StatusCode < 300 {
		return nil, errors.New("invalid status code '" + res.Status + "'")
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func extractLibName(s string) (lib, name string) {
	if ind := strings.IndexAny(s, `\/`); ind == -1 {
		name = s
	} else {
		name, lib = s[:ind], s[ind+1:]
	}
	return
}

func doLibUse(s Settings) error {
	var (
		found     ManifestItem
		lib, name = extractLibName(s.arg)
	)

	if lib != "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}

		content, err := os.ReadFile(filepath.Join(configDir, "wednesday", "trusted", lib+".json"))
		if err != nil {
			return err
		}

		manifest := make(map[string]ManifestItem)
		if err = json.Unmarshal(content, &manifest); err != nil {
			return err
		}

		ok := false
		if found, ok = manifest[name]; !ok {
			return errors.New("cannot find '" + name + "' on '" + lib + "' library")
		}
	} else {
		coll, err := LoadCollection()
		if err != nil {
			return err
		}

		if lib, name, found, err = coll.Find(name); err != nil {
			return err
		}
	}

	return found.Download(s.InputDir, lib, name)
}
