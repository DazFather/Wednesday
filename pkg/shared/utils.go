package shared

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func getBody(link string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Get(link)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.New("invalid status code '" + res.Status + "'")
	}

	return io.ReadAll(res.Body)
}

func CleanURL(link string) (string, error) {
	u, err := url.ParseRequestURI(link)
	if err != nil {
		return "", err
	}
	return u.RequestURI(), nil
}

func FetchContent(link string) (content []byte, err error) {
	if url, e := CleanURL(link); e == nil {
		return getBody(url)
	}
	return os.ReadFile(link) // #nosec G304 -- reads an explicit local path/URL target provided by trusted templates/build inputs
}

func Compact[T comparable](list []T) []T {
	var res []T
	for _, item := range list {
		if !slices.Contains(res, item) {
			res = append(res, item)
		}
	}
	return res
}

func IsOnDir(path, dir string) bool {
	var err error
	if path, err = filepath.Abs(path); err != nil {
		panic(err)
	}
	if dir, err = filepath.Abs(dir); err != nil {
		panic(err)
	}

	rel, err := filepath.Rel(dir, path)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}
