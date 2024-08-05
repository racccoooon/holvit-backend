package repos

import (
	"github.com/google/uuid"
	"holvit/h"
)

type BaseFilter struct {
	Id         h.Optional[uuid.UUID]
	PagingInfo h.Optional[PagingInfo]
}

type PagingInfo struct {
	PageSize   int
	PageNumber int
}

type FilterResult[T any] interface {
	Values() []T
	Count() int
	First() h.Optional[T]
}

func first[T any](r FilterResult[T]) h.Optional[T] {
	values := r.Values()
	if len(values) == 0 {
		return h.None[T]()
	}
	return h.Some(values[0])
}

type pagedResult[T any] struct {
	values []T
	count  int
}

func (p *pagedResult[T]) Values() []T {
	return p.values
}

func (p *pagedResult[T]) Count() int {
	return p.count
}

func (p *pagedResult[T]) First() h.Optional[T] {
	return first[T](p)
}

func (p pagedResult[T]) ToResult() FilterResult[T] {
	return &p
}

func NewPagingInfo(pageSize int, pageNumber int) PagingInfo {
	return PagingInfo{
		PageSize:   pageSize,
		PageNumber: pageNumber,
	}
}
