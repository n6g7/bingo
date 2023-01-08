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

func (s *Set[T]) Contains(key T) bool {
	val, found := s.m[key]
	return found && val
}

func (s *Set[T]) Inter(other *Set[T]) (inter *Set[T]) {
	var small, big *Set[T]
	if s.Length() < other.Length() {
		small = s
		big = other
	} else {
		small = other
		big = s
	}
	inter = NewSet[T]()
	for v := range small.Iter() {
		if big.Contains(v) {
			inter.Add(v)
		}
	}
	return
}

func (s *Set[T]) Union(other *Set[T]) (union *Set[T]) {
	union = NewSet[T]()
	for v := range s.Iter() {
		union.Add(v)
	}
	for v := range other.Iter() {
		union.Add(v)
	}
	return
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
