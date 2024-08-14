package sqlb

import (
	"fmt"
	"github.com/DataDog/go-sqllexer"
	"github.com/sourcegraph/conc/iter"
	"holvit/h"
	"strconv"
	"strings"
)

type SqlQuery struct {
	Query      string
	Parameters []any
}

type Query interface{}

type SelectQuery interface {
	Select(cols ...any) SelectQuery
	Distinct(on ...any) SelectQuery
	From(table any) SelectQuery
	Join(table any, on any, params ...any) SelectQuery
	InnerJoin(table any, on any, params ...any) SelectQuery
	LeftJoin(table any, on any, params ...any) SelectQuery
	RightJoin(table any, on any, params ...any) SelectQuery
	FullJoin(table any, on any, params ...any) SelectQuery
	CrossJoin(table any) SelectQuery
	Where(condition any, params ...any) SelectQuery
	OrderBy(fields ...any) SelectQuery
	Limit(limit any) SelectQuery
	Offset(offset any) SelectQuery
	LockForUpdate(skipLocked bool) SelectQuery

	Build() SqlQuery
}

type selectQuery struct {
	columns       []any
	from          []any
	joins         []selectJoin
	where         []QueryFragment
	distinct      bool
	distinctOn    []QueryFragment
	orderBy       []QueryFragment
	limit         h.Opt[QueryFragment]
	offset        h.Opt[QueryFragment]
	lockForUpdate bool
	skipLocked    bool
	with          []with
}

type joinType int

const (
	joinDefault joinType = iota
	joinInner
	joinLeftOuter
	joinRightOuter
	joinFullOuter
	joinCross
)

type selectJoin struct {
	type_     joinType
	table     any
	condition QueryFragment
}

func (s *selectQuery) Select(cols ...any) SelectQuery {
	s.columns = append(s.columns, cols...)
	return s
}

func (s *selectQuery) From(table any) SelectQuery {
	s.from = append(s.from, table)
	return s
}

func (s *selectQuery) Where(condition any, params ...any) SelectQuery {
	s.where = append(s.where, Term(condition, params...))
	return s
}

func (s *selectQuery) Limit(limit any) SelectQuery {
	if num, ok := limit.(int); ok {
		s.limit = h.Some(Term("?", num))
	} else {
		s.limit = h.Some(Term(limit))
	}
	return s
}

func (s *selectQuery) Offset(offset any) SelectQuery {
	if num, ok := offset.(int); ok {
		s.offset = h.Some(Term("?", num))
	} else {
		s.offset = h.Some(Term(offset))
	}
	return s
}

func (s *selectQuery) Distinct(on ...any) SelectQuery {
	s.distinct = true
	s.distinctOn = append(s.distinctOn, iter.Map(on, func(x *any) QueryFragment { return Term(*x) })...)
	return s
}

func (s *selectQuery) LockForUpdate(skipLocked bool) SelectQuery {
	s.lockForUpdate = true
	s.skipLocked = skipLocked
	return s
}

func (s *selectQuery) OrderBy(fields ...any) SelectQuery {
	s.orderBy = append(s.orderBy, iter.Map(fields, func(x *any) QueryFragment { return Term(*x) })...)
	return s
}

func (s *selectQuery) Join(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinDefault,
		table:     table,
		condition: Term(on, params...),
	})
	return s
}

func (s *selectQuery) InnerJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinInner,
		table:     table,
		condition: Term(on, params...),
	})
	return s
}

func (s *selectQuery) LeftJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinLeftOuter,
		table:     table,
		condition: Term(on, params...),
	})
	return s
}

func (s *selectQuery) RightJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinRightOuter,
		table:     table,
		condition: Term(on, params...),
	})
	return s
}

func (s *selectQuery) FullJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinFullOuter,
		table:     table,
		condition: Term(on, params...),
	})
	return s
}

func (s *selectQuery) CrossJoin(table any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_: joinCross,
		table: table,
	})
	return s
}

func (s *selectQuery) buildWith(sql *strings.Builder, p *params) {
	if len(s.with) == 0 {
		return
	}
	withParts := make([]string, 0, len(s.with))
	for _, with := range s.with {
		withParts = append(withParts, fmt.Sprintf("%s AS %s", with.name, with.query.build(p)))
	}
	sql.WriteString("WITH ")
	sql.WriteString(strings.Join(withParts, ", "))
	sql.WriteString(" ")
}

func (s *selectQuery) buildSelect(sql *strings.Builder, p *params) {
	var colStrings []string
	var buildSelectCol func(col any) string
	buildSelectCol = func(col any) string {
		if as, ok := col.(queryAs); ok {
			str := buildSelectCol(as.query)
			return fmt.Sprintf("%s AS %s", str, as.name)
		} else if s, ok := col.(string); ok {
			return s
		} else if q, ok := col.(QueryFragment); ok {
			return q.build(p)
		} else if q, ok := col.(*selectQuery); ok {
			return fmt.Sprintf("(%s)", q.build(p))
		} else {
			panic("unsupported type")
		}
	}

	for _, col := range s.columns {
		colStrings = append(colStrings, buildSelectCol(col))
	}

	sql.WriteString("SELECT ")
	if s.distinct {
		sql.WriteString("DISTINCT ")
		if len(s.distinctOn) > 0 {
			onParts := make([]string, 0, len(s.distinctOn))
			for _, on := range s.distinctOn {
				onParts = append(onParts, on.build(p))
			}
			sql.WriteString("ON (")
			sql.WriteString(strings.Join(onParts, ", "))
			sql.WriteString(") ")
		}
	}
	sql.WriteString(strings.Join(colStrings, ", "))
}

func (s *selectQuery) buildFrom(sql *strings.Builder, p *params) {
	if len(s.from) == 0 {
		return
	}
	var froms []string
	for _, from := range s.from {
		if s, ok := from.(string); ok {
			froms = append(froms, s)
		} else if q, ok := from.(*selectQuery); ok {
			query := q.build(p)
			froms = append(froms, fmt.Sprintf("(%s)", query))
		}
	}
	sql.WriteString(fmt.Sprintf(" FROM %s", strings.Join(froms, ", ")))
}

func (s *selectQuery) buildJoin(sql *strings.Builder, p *params) {
	for _, join := range s.joins {
		switch join.type_ {
		case joinDefault:
			sql.WriteString(fmt.Sprintf(" JOIN %s ON %s", join.table, join.condition.build(p)))
		case joinInner:
			sql.WriteString(fmt.Sprintf(" INNER JOIN %s ON %s", join.table, join.condition.build(p)))
		case joinLeftOuter:
			sql.WriteString(fmt.Sprintf(" LEFT OUTER JOIN %s ON %s", join.table, join.condition.build(p)))
		case joinRightOuter:
			sql.WriteString(fmt.Sprintf(" RIGHT OUTER JOIN %s ON %s", join.table, join.condition.build(p)))
		case joinFullOuter:
			sql.WriteString(fmt.Sprintf(" FULL OUTER JOIN %s ON %s", join.table, join.condition.build(p)))
		case joinCross:
			sql.WriteString(fmt.Sprintf(" CROSS JOIN %s", join.table))
		}
	}
}

func (s *selectQuery) buildWhere(sql *strings.Builder, p *params) {
	wrapConds := len(s.where) > 1
	for idx, where := range s.where {
		if idx == 0 {
			sql.WriteString(" WHERE ")
		} else {
			sql.WriteString(" AND ")
		}
		if wrapConds {
			sql.WriteString("(")
		}
		sql.WriteString(where.build(p))
		if wrapConds {
			sql.WriteString(")")
		}
	}
}

func (s *selectQuery) buildOrderBy(sql *strings.Builder, p *params) {
	if len(s.orderBy) == 0 {
		return
	}
	sql.WriteString(" ORDER BY ")
	orderParts := make([]string, 0, len(s.orderBy))
	for _, orderBy := range s.orderBy {
		orderParts = append(orderParts, orderBy.build(p))
	}
	sql.WriteString(strings.Join(orderParts, ", "))
}

func (s *selectQuery) buildLimitOffset(sql *strings.Builder, p *params) {
	if limit, ok := s.limit.Get(); ok {
		sql.WriteString(" LIMIT ")
		sql.WriteString(limit.build(p))
	}
	if offset, ok := s.offset.Get(); ok {
		sql.WriteString(" OFFSET ")
		sql.WriteString(offset.build(p))
	}
}

func (s *selectQuery) buildLock(sql *strings.Builder, p *params) {
	if s.lockForUpdate {
		sql.WriteString(" FOR UPDATE")
		if s.skipLocked {
			sql.WriteString(" SKIP LOCKED")
		}
	}
}

type params []any

func (p *params) append(param any) string {
	*p = append(*p, param)
	return "$" + strconv.Itoa(len(*p))
}

func (s *selectQuery) build(p *params) string {
	var sql strings.Builder
	s.buildWith(&sql, p)
	s.buildSelect(&sql, p)
	s.buildFrom(&sql, p)
	s.buildJoin(&sql, p)
	s.buildWhere(&sql, p)
	s.buildOrderBy(&sql, p)
	s.buildLimitOffset(&sql, p)
	s.buildLock(&sql, p)
	return sql.String()
}

func (s *selectQuery) Build() SqlQuery {
	var p params
	sql := s.build(&p)
	return SqlQuery{
		Query:      sql,
		Parameters: p,
	}
}

type WithQuery interface {
	With(name string, query any, params ...any) WithQuery
	Select(args ...any) SelectQuery
}

type QueryAs interface {
}

type queryAs struct {
	query any
	name  string
}

func As(query any, name string) queryAs {
	return queryAs{
		query: query,
		name:  name,
	}
}

type with struct {
	name  string
	query QueryFragment
}

type withQuery struct {
	withs []with
}

func (w *withQuery) With(name string, query any, params ...any) WithQuery {
	if s, ok := query.(string); ok {
		query = "(" + s + ")"
	}
	w.withs = append(w.withs, with{name: name, query: Term(query, params...)})
	return w
}

func (w *withQuery) Select(cols ...any) SelectQuery {
	q := &selectQuery{
		with: w.withs,
	}
	return q.Select(cols...)
}

func With(name string, query any, params ...any) WithQuery {
	if s, ok := query.(string); ok {
		query = "(" + s + ")"
	}
	return &withQuery{
		withs: []with{{name: name, query: Term(query, params...)}},
	}
}

func Select(col any, cols ...any) SelectQuery {
	c := make([]any, 0, len(cols)+1)
	c = append(c, col)
	c = append(c, cols...)
	return &selectQuery{columns: c}

}

type QueryFragment struct {
	term   string
	params []any
}

func (t *QueryFragment) build(p *params) string {
	lexer := sqllexer.New(t.term)
	tokens := lexer.ScanAll()
	var parts []string
	paramIdx := 0
	for _, token := range tokens {
		if token.Type == sqllexer.OPERATOR && token.Value == "?" {
			param := t.params[paramIdx]
			if q, ok := param.(*selectQuery); ok {
				parts = append(parts, "("+q.build(p)+")")
			} else if q, ok := param.(QueryFragment); ok {
				parts = append(parts, q.build(p))
			} else {
				parts = append(parts, p.append(param))
			}
			paramIdx++
		} else {
			parts = append(parts, token.Value)
		}
	}
	return strings.Join(parts, "")
}

func Term(term any, params ...any) QueryFragment {
	if s, ok := term.(string); ok {
		return QueryFragment{term: s, params: params}
	}
	if len(params) != 0 {
		panic(fmt.Errorf("term can only take params if it is a string"))
	}
	if q, ok := term.(QueryFragment); ok {
		return q
	} else if q, ok := term.(SelectQuery); ok {
		return QueryFragment{
			term:   "?",
			params: []any{q},
		}
	} else {
		panic("unsupported type")
	}
}

func andor(joiner string, terms ...any) QueryFragment {
	if len(terms) == 0 {
		panic("no terms given")
	}
	if len(terms) == 1 {
		return Term(terms[0])
	}
	parts := make([]string, len(terms))
	params := make([]any, len(terms))
	for i, term := range terms {
		parts[i] = "(?)"
		if s, ok := term.(string); ok {
			params[i] = Term(s)
		} else {
			params[i] = term
		}
	}
	return QueryFragment{
		term:   strings.Join(parts, joiner),
		params: params,
	}
}

func And(terms ...any) QueryFragment {
	return andor(" AND ", terms...)
}
func Or(terms ...any) QueryFragment {
	return andor(" OR ", terms...)
}
func Not(term any) QueryFragment {
	var param any
	if s, ok := term.(string); ok {
		param = Term(s)
	} else {
		param = term
	}
	return QueryFragment{
		term:   "NOT(?)",
		params: []any{param},
	}
}

func Exists(subquery SelectQuery) QueryFragment {
	return QueryFragment{
		term:   "EXISTS ?",
		params: []any{subquery},
	}
}
