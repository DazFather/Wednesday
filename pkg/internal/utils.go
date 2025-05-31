package internal

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

func cleanURL(link string) (string, error) {
	u, err := url.ParseRequestURI(link)
	if err != nil {
		return "", err
	}
	return u.RequestURI(), nil
}

func FetchContent(link string) (content []byte, err error) {
	if url, e := cleanURL(link); e == nil {
		return getBody(url)
	}
	return os.ReadFile(link)
}
