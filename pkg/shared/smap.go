package shared

import (
	"encoding/xml"
	"path/filepath"
	"strings"
	"sync"
)

type Smap[K comparable, V any] sync.Map

func (s *Smap[K, V]) Load(key K) (V, bool) {
	v, ok := (*sync.Map)(s).Load(key)
	return v.(V), ok
}

func (s *Smap[K, V]) Store(key K, val V) {
	(*sync.Map)(s).Store(key, val)
}

func (s *Smap[K, V]) Delete(key K) {
	(*sync.Map)(s).Delete(key)
}

func (s *Smap[K, V]) Range(fn func(K, V) bool) {
	(*sync.Map)(s).Range(func(rk, rv any) bool {
		return fn(rk.(K), rv.(V))
	})
}

func splitExt(name string) (base, ext string) {
	ext = strings.ToLower(filepath.Ext(name))
	base = name[:len(name)-len(ext)]
	if ext == ".html" {
		ext = strings.ToLower(filepath.Ext(base)) + ext
		base = name[:len(name)-len(ext)]
	}
	return
}

func validHTML(content string) error {
	var dec = xml.NewDecoder(strings.NewReader(content))
	dec.Strict = false
	return dec.Decode(new(interface{}))
}
