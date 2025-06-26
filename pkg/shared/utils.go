package shared

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
)

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
	return os.ReadFile(link)
}

func Compact[T comparable](list []T) []T {
	m := make(map[T]bool)
	for _, item := range list {
		if found, _ := m[item]; !found {
			m[item] = true
		}
	}

	i, res := 0, make([]T, len(m))
	for item := range m {
		res[i] = item
		i++
	}
	return res
}
