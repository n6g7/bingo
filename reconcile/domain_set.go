package reconcile

type Set[T comparable] struct {
	m map[T]bool
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{make(map[T]bool)}
}

func (s *Set[T]) Length() int {
	return len(s.m)
}

func (s *Set[T]) Add(val T) {
	s.m[val] = true
}

func (s *Set[T]) Iter() chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		for v := range s.m {
			ch <- v
		}
	}()
	return ch
}

func (s *Set[T]) Diff(other *Set[T]) *Set[T] {
	diff := NewSet[T]()
	for v := range s.m {
		if _, found := other.m[v]; !found {
			diff.Add(v)
		}
	}
	return diff
}

type DomainSet = Set[string]

func NewDomainSet() *DomainSet {
	return NewSet[string]()
}
