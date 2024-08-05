package repos

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
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

func (i PagingInfo) Apply(sb *sqlbuilder.SelectBuilder) {
	sb.Limit(i.PageSize).Offset(i.PageSize * (i.PageNumber - 1))
}

func (i PagingInfo) SqlString() string {
	return fmt.Sprintf(" limit %d offset %d", i.PageSize, i.PageSize*(i.PageNumber-1))
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
	values     []T
	totalCount int
}

func NewPagedResult[T any](values []T, totalCount int) FilterResult[T] {
	return &pagedResult[T]{
		values:     values,
		totalCount: totalCount,
	}
}

func (p *pagedResult[T]) Values() []T {
	return p.values
}

func (p *pagedResult[T]) Count() int {
	return p.totalCount
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
