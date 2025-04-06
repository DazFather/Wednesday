package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

	if m.checksum != "" {
		if fmt.Sprintf("%x", sha256.Sum256(content)) != m.checksum {
			return fmt.Errorf("failed checksum validation when downloading %q component", lib+"/"+name)
		}
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
		err = fmt.Errorf("cannot find any component named %q on trusted libraries", name)
	case 1:
		selected = manifests[results[0]][name]
	default:
		err = fmt.Errorf("%d matching components on different libraries: %q", len(results), strings.Join(results, ", "))
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
	if s.arg == "" {
		return fmt.Errorf("missing manifest reference")
	}

	content, err := getContent(s.arg)
	if err != nil {
		return
	}

	trusted, err := getWedConfigDir("trusted")
	if err != nil {
		return
	}

	var manifest = make(map[string]ManifestItem)
	if err = json.Unmarshal(content, &manifest); err != nil {
		return
	}

	if s.download {
		if err = os.MkdirAll(filepath.Join(trusted, s.name), os.ModePerm); err != nil {
			return
		}

		for name, c := range manifest {
			if err = c.Download(trusted, s.name, name); err != nil {
				return
			}
			c.download = filepath.Join(trusted, s.name, name+".wed.html")
		}
	}

	manifestPath := filepath.Join(trusted, s.name+".json")
	if err = os.WriteFile(manifestPath, content, 0666); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(trusted, os.ModePerm); err == nil {
			err = os.WriteFile(manifestPath, content, 0666)
		}
	}

	return
}

func doLibUntrust(s Settings) error {
	filename, err := getWedConfigDir("trusted", s.arg+".json")
	if err == nil {
		err = os.Remove(filename)
	}

	return err
}

func doLibUse(s Settings) error {
	var (
		found     ManifestItem
		lib, name = extractLibName(s.arg)
	)

	if lib != "" {
		filename, err := getWedConfigDir("trusted", lib+".json")
		if err != nil {
			return err
		}

		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		manifest := make(map[string]ManifestItem)
		if err = json.Unmarshal(content, &manifest); err != nil {
			return err
		}

		ok := false
		if found, ok = manifest[name]; !ok {
			return fmt.Errorf("cannot find %q on library %q", name, lib)
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
