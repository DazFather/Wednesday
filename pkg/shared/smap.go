package shared

import (
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
