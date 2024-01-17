package reconcile

import "github.com/n6g7/bingo/internal/set"

type DomainSet = set.Set[string]

func NewDomainSet() *DomainSet {
	return set.NewSet([]string{})
}
