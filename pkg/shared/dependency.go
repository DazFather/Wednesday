package shared

import (
	"iter"
	"slices"
	"sync"
)

type pile[T any] []T

func (p *pile[T]) push(items ...T) {
	*p = append(*p, items...)
}

func (p *pile[T]) pop(items ...T) T {
	res := (*p)[len(*p)-1]
	*p = (*p)[:len(*p)-1]
	return res
}

func (p *pile[T]) find(fn func(T) bool) bool {
	for _, item := range *p {
		if fn(item) {
			return true
		}
	}
	return false
}

func (p *pile[T]) iter() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		list := []T(*p)
		for i := range list {
			if !yield(i, list[len(list)-1-i]) {
				return
			}
		}
	}
}

type Identifiable[T comparable] interface {
	Identifier() T
}

type Dependency[S comparable, T Identifiable[S]] struct {
	Data    T
	Imports []Dependency[S, T]
}

func pileDeps[S comparable, T Identifiable[S]](p *pile[[]Dependency[S, T]], deps []Dependency[S, T]) *pile[[]Dependency[S, T]] {
	if len(deps) == 0 {
		return p
	}
	p.push(deps)
	for _, dep := range deps {
		pileDeps(p, dep.Imports)
	}
	return p
}

func checkCircular[S comparable, T Identifiable[S]](stack pile[S], current Dependency[S, T]) *Dependency[S, T] {
	stack.push(current.Data.Identifier())
	defer stack.pop()

	for _, imprt := range current.Imports {
		if stack.find(func(v S) bool { return v == imprt.Data.Identifier() }) {
			return &imprt
		}
		if circ := checkCircular(stack, imprt); circ != nil {
			return circ
		}
	}
	return nil
}

func HasCircularDep[S comparable, T Identifiable[S]](deps []Dependency[S, T]) []S {
	var (
		ch = make(chan S)
		wg sync.WaitGroup
	)

	wg.Add(len(deps))
	for _, dep := range deps {
		go func() {
			defer wg.Done()
			if circ := checkCircular(nil, dep); circ != nil {
				ch <- circ.Data.Identifier()
			}
		}()
	}
	wg.Wait()
	close(ch)

	var arr []S
	for circularID := range ch {
		if !slices.Contains(arr, circularID) {
			arr = append(arr, circularID)
		}
	}
	return arr
}

func Inverse[S comparable, T Identifiable[S]](deps []Dependency[S, T]) iter.Seq2[int, Dependency[S, T]] {
	return func(yield func(int, Dependency[S, T]) bool) {
		var stack = pileDeps(new(pile[[]Dependency[S, T]]), deps)

		for i, list := range stack.iter() {
			for _, dep := range list {
				if !yield(i, dep) {
					return
				}
			}
		}
	}
}
