package repos

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"holvit/h"
	"holvit/sqlb"
)

type BaseFilter struct {
	Id         h.Opt[uuid.UUID]
	PagingInfo h.Opt[PagingInfo]
	SortInfo   h.Opt[SortInfo]
	SearchText h.Opt[string]
}

func (f *BaseFilter) CountCol() string {
	if f.PagingInfo.IsSome() {
		return "count(*) over()"
	}
	return "-1"
}

type SortInfo struct {
	Field     string
	Ascending bool
}

func (i SortInfo) Apply(sb *sqlbuilder.SelectBuilder) {
	sb.OrderBy(i.Field)
	if i.Ascending {
		sb.Asc()
	} else {
		sb.Desc()
	}
}

func (i SortInfo) Apply2(q sqlb.SelectQuery) {
	field := i.Field
	if i.Ascending {
		field += " asc"
	} else {
		field += " desc"
	}
	q.OrderBy(field)
}

func (i SortInfo) SqlString() string {
	direction := " asc"
	if !i.Ascending {
		direction = " desc"
	}
	return fmt.Sprintf(" order by %s%s", i.Field, direction)
}

type PagingInfo struct {
	PageSize   int
	PageNumber int
}

func (i PagingInfo) Apply(sb *sqlbuilder.SelectBuilder) {
	sb.Limit(i.PageSize).Offset(i.PageSize * (i.PageNumber - 1))
}

func (i PagingInfo) Apply2(sb sqlb.SelectQuery) {
	sb.Limit(i.PageSize).Offset(i.PageSize * (i.PageNumber - 1))
}

func (i PagingInfo) SqlString() string {
	return fmt.Sprintf(" limit %d offset %d", i.PageSize, i.PageSize*(i.PageNumber-1))
}

type FilterResult[T any] interface {
	Values() []T
	Count() int
	First() T
	FirstOrNone() h.Opt[T]
	Single() T
	SingleOrNone() h.Opt[T]
	Any() bool
}

func first[T any](r FilterResult[T]) h.Opt[T] {
	values := r.Values()
	if len(values) == 0 {
		return h.None[T]()
	}
	return h.Some(values[0])
}

func single[T any](r FilterResult[T]) h.Opt[T] {
	values := r.Values()
	if len(values) == 1 {
		return h.Some(values[0])
	} else if len(values) == 0 {
		return h.None[T]()
	}

	panic(errors.New("too many values"))
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

func (p *pagedResult[T]) Any() bool {
	return len(p.values) > 0
}

func (p *pagedResult[T]) First() T {
	return first[T](p).Unwrap()
}

func (p *pagedResult[T]) FirstOrNone() h.Opt[T] {
	return first[T](p)
}

func (p *pagedResult[T]) Single() T {
	return single[T](p).Unwrap()
}

func (p *pagedResult[T]) SingleOrNone() h.Opt[T] {
	return single[T](p)
}

func NewPagingInfo(pageSize int, pageNumber int) PagingInfo {
	return PagingInfo{
		PageSize:   pageSize,
		PageNumber: pageNumber,
	}
}
