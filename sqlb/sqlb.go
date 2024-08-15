package sqlb

import (
	"fmt"
	"github.com/DataDog/go-sqllexer"
	"holvit/h"
	"strconv"
	"strings"
)

type SqlQuery struct {
	Query      string
	Parameters []any
}

type Query interface {
	Build() SqlQuery
	build(sql *strings.Builder, p *params)
}

type SelectQuery interface {
	Query

	Select(cols ...any) SelectQuery
	Distinct(on ...any) SelectQuery
	From(table any) SelectQuery
	FromAs(table any, name string) SelectQuery
	Join(table any, on any, params ...any) SelectQuery
	JoinAs(table any, name string, on any, params ...any) SelectQuery
	InnerJoin(table any, on any, params ...any) SelectQuery
	InnerJoinAs(table any, name string, on any, params ...any) SelectQuery
	LeftJoin(table any, on any, params ...any) SelectQuery
	LeftJoinAs(table any, name string, on any, params ...any) SelectQuery
	RightJoin(table any, on any, params ...any) SelectQuery
	RightJoinAs(table any, name string, on any, params ...any) SelectQuery
	FullJoin(table any, on any, params ...any) SelectQuery
	FullJoinAs(table any, name string, on any, params ...any) SelectQuery
	CrossJoin(table any) SelectQuery
	CrossJoinAs(table any, name string) SelectQuery
	Where(condition any, params ...any) SelectQuery
	GroupBy(groupingElements ...any) SelectQuery
	Having(condition any, params ...any) SelectQuery
	OrderBy(fields ...any) SelectQuery
	Limit(limit any) SelectQuery
	Offset(offset any) SelectQuery
	LockForUpdate(skipLocked bool) SelectQuery
}

type selectQuery struct {
	columns       []RawQuery
	from          []RawQuery
	joins         []selectJoin
	where         []RawQuery
	groupBy       []RawQuery
	having        []RawQuery
	distinct      bool
	distinctOn    []RawQuery
	orderBy       []RawQuery
	limit         h.Opt[RawQuery]
	offset        h.Opt[RawQuery]
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
	table     RawQuery
	as        h.Opt[string]
	condition RawQuery
}

func (s *selectQuery) Select(cols ...any) SelectQuery {
	s.columns = append(s.columns, makeRawFragments(cols)...)
	return s
}

func (s *selectQuery) From(table any) SelectQuery {
	s.from = append(s.from, makeRawFragment(table))
	return s
}

func (s *selectQuery) FromAs(table any, name string) SelectQuery {
	s.from = append(s.from, As(table, name))
	return s
}

func (s *selectQuery) Where(condition any, params ...any) SelectQuery {
	s.where = append(s.where, makeRawFragment(condition, params...))
	return s
}

func (s *selectQuery) GroupBy(groupingElements ...any) SelectQuery {
	s.groupBy = append(s.groupBy, makeRawFragments(groupingElements)...)
	return s
}

func (s *selectQuery) Having(condition any, params ...any) SelectQuery {
	s.having = append(s.having, makeRawFragment(condition, params...))
	return s
}

func (s *selectQuery) Limit(limit any) SelectQuery {
	if num, ok := limit.(int); ok {
		s.limit = h.Some(makeRawFragment("?", num))
	} else {
		s.limit = h.Some(makeRawFragment(limit))
	}
	return s
}

func (s *selectQuery) Offset(offset any) SelectQuery {
	if num, ok := offset.(int); ok {
		s.offset = h.Some(makeRawFragment("?", num))
	} else {
		s.offset = h.Some(makeRawFragment(offset))
	}
	return s
}

func (s *selectQuery) Distinct(on ...any) SelectQuery {
	s.distinct = true
	s.distinctOn = append(s.distinctOn, makeRawFragments(on)...)
	return s
}

func (s *selectQuery) LockForUpdate(skipLocked bool) SelectQuery {
	s.lockForUpdate = true
	s.skipLocked = skipLocked
	return s
}

func (s *selectQuery) OrderBy(fields ...any) SelectQuery {
	s.orderBy = append(s.orderBy, makeRawFragments(fields)...)
	return s
}

func (s *selectQuery) Join(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinDefault,
		table:     makeRawFragment(table),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) JoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinDefault,
		table:     makeRawFragment(table),
		as:        h.Some(name),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) InnerJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinInner,
		table:     makeRawFragment(table),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) InnerJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinInner,
		table:     makeRawFragment(table),
		as:        h.Some(name),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) LeftJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinLeftOuter,
		table:     makeRawFragment(table),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) LeftJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinLeftOuter,
		table:     makeRawFragment(table),
		as:        h.Some(name),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) RightJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinRightOuter,
		table:     makeRawFragment(table),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) RightJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinRightOuter,
		table:     makeRawFragment(table),
		as:        h.Some(name),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) FullJoin(table any, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinFullOuter,
		table:     makeRawFragment(table),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) FullJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_:     joinFullOuter,
		table:     makeRawFragment(table),
		as:        h.Some(name),
		condition: makeRawFragment(on, params...),
	})
	return s
}

func (s *selectQuery) CrossJoin(table any) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_: joinCross,
		table: makeRawFragment(table),
	})
	return s
}

func (s *selectQuery) CrossJoinAs(table any, name string) SelectQuery {
	s.joins = append(s.joins, selectJoin{
		type_: joinCross,
		as:    h.Some(name),
		table: makeRawFragment(table),
	})
	return s
}

func (s *selectQuery) buildWith(sql *strings.Builder, p *params) {
	if len(s.with) == 0 {
		return
	}
	sql.WriteString("WITH ")
	for i, with := range s.with {
		sql.WriteString(with.name)
		sql.WriteString(" AS ")
		with.query.build(sql, p)
		if i < len(s.with)-1 {
			sql.WriteString(", ")
		}
	}
	sql.WriteString(" ")
}

func buildFragments(sql *strings.Builder, p *params, fragments []RawQuery, joiner string) {
	for i, fragment := range fragments {
		fragment.build(sql, p)
		if i < len(fragments)-1 {
			sql.WriteString(joiner)
		}
	}
}

func (s *selectQuery) buildSelect(sql *strings.Builder, p *params) {
	sql.WriteString("SELECT ")

	if s.distinct {
		sql.WriteString("DISTINCT ")
		if len(s.distinctOn) > 0 {
			sql.WriteString("ON (")
			buildFragments(sql, p, s.distinctOn, ", ")
			sql.WriteString(") ")
		}
	}
	buildFragments(sql, p, s.columns, ", ")
}

func (s *selectQuery) buildFrom(sql *strings.Builder, p *params) {
	if len(s.from) == 0 {
		return
	}
	sql.WriteString(" FROM ")
	buildFragments(sql, p, s.from, ", ")
}

func (s *selectQuery) buildJoin(sql *strings.Builder, p *params) {
	for _, join := range s.joins {
		switch join.type_ {
		case joinDefault:
			sql.WriteString(" JOIN ")
		case joinInner:
			sql.WriteString(" INNER JOIN ")
		case joinLeftOuter:
			sql.WriteString(" LEFT OUTER JOIN ")
		case joinRightOuter:
			sql.WriteString(" RIGHT OUTER JOIN ")
		case joinFullOuter:
			sql.WriteString(" FULL OUTER JOIN ")
		case joinCross:
			sql.WriteString(" CROSS JOIN ")
		}

		join.table.build(sql, p)

		if name, ok := join.as.Get(); ok {
			sql.WriteString(" AS ")
			sql.WriteString(name)
		}

		if join.type_ != joinCross {
			sql.WriteString(" ON ")
			join.condition.build(sql, p)
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
		where.build(sql, p)
		if wrapConds {
			sql.WriteString(")")
		}
	}
}

func (s *selectQuery) buildGroupBy(sql *strings.Builder, p *params) {
	if len(s.groupBy) == 0 {
		return
	}
	sql.WriteString(" GROUP BY ")
	buildFragments(sql, p, s.groupBy, ", ")
}

func (s *selectQuery) buildHaving(sql *strings.Builder, p *params) {
	wrapConds := len(s.having) > 1
	for idx, having := range s.having {
		if idx == 0 {
			sql.WriteString(" HAVING ")
		} else {
			sql.WriteString(" AND ")
		}
		if wrapConds {
			sql.WriteString("(")
		}
		having.build(sql, p)
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
	buildFragments(sql, p, s.orderBy, ", ")
}

func (s *selectQuery) buildLimitOffset(sql *strings.Builder, p *params) {
	if limit, ok := s.limit.Get(); ok {
		sql.WriteString(" LIMIT ")
		limit.build(sql, p)
	}
	if offset, ok := s.offset.Get(); ok {
		sql.WriteString(" OFFSET ")
		offset.build(sql, p)
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

func (s *selectQuery) build(sql *strings.Builder, p *params) {
	s.buildWith(sql, p)
	s.buildSelect(sql, p)
	s.buildFrom(sql, p)
	s.buildJoin(sql, p)
	s.buildWhere(sql, p)
	s.buildGroupBy(sql, p)
	s.buildHaving(sql, p)
	s.buildOrderBy(sql, p)
	s.buildLimitOffset(sql, p)
	s.buildLock(sql, p)
}

func (s *selectQuery) Build() SqlQuery {
	var p params
	var sql strings.Builder
	s.build(&sql, &p)
	return SqlQuery{
		Query:      sql.String(),
		Parameters: p,
	}
}

type WithQuery interface {
	With(name string, query any, params ...any) WithQuery
	Select(args ...any) SelectQuery
	InsertInto(table string, columns ...string) InsertQuery
}

func As(query any, name string) RawQuery {
	return &rawQuery{
		term:   "? AS " + name,
		params: []any{makeRawFragment(query)},
	}
}

type with struct {
	name  string
	query RawQuery
}

type withQuery struct {
	withs []with
}

func (w *withQuery) With(name string, query any, params ...any) WithQuery {
	if s, ok := query.(string); ok {
		query = "(" + s + ")"
	}
	w.withs = append(w.withs, with{name: name, query: makeRawFragment(query, params...)})
	return w
}

func (w *withQuery) Select(cols ...any) SelectQuery {
	q := &selectQuery{
		with: w.withs,
	}
	return q.Select(cols...)
}

func (w *withQuery) InsertInto(table string, columns ...string) InsertQuery {
	//TODO implement me
	panic("implement me")
}

func With(name string, query any, params ...any) WithQuery {
	if s, ok := query.(string); ok {
		query = "(" + s + ")"
	}
	return &withQuery{
		withs: []with{{name: name, query: makeRawFragment(query, params...)}},
	}
}

func Select(col any, cols ...any) SelectQuery {
	c := make([]any, 0, len(cols)+1)
	c = append(c, col)
	c = append(c, cols...)
	return &selectQuery{columns: makeRawFragments(c)}

}

type RawQuery interface {
	Query
}

type rawQuery struct {
	term   string
	params []any
}

func (q *rawQuery) build(sql *strings.Builder, p *params) {
	lexer := sqllexer.New(q.term)
	tokens := lexer.ScanAll()
	paramIdx := 0
	for _, token := range tokens {
		if token.Type == sqllexer.OPERATOR && strings.ContainsRune(token.Value, '?') {
			for _, r := range token.Value {
				if r == '?' {
					param := q.params[paramIdx]
					if q, ok := param.(*selectQuery); ok {
						sql.WriteRune('(')
						q.build(sql, p)
						sql.WriteRune(')')
					} else if q, ok := param.(*rawQuery); ok {
						q.build(sql, p)
					} else {
						sql.WriteString(p.append(param))
					}
					paramIdx++
				} else {
					sql.WriteRune(r)
				}
			}
		} else {
			sql.WriteString(token.Value)
		}
	}
}

func (q *rawQuery) Build() SqlQuery {
	var p params
	var sql strings.Builder
	q.build(&sql, &p)
	return SqlQuery{
		Query:      sql.String(),
		Parameters: p,
	}
}

func makeRawFragments(input []any) []RawQuery {
	fragments := make([]RawQuery, len(input))
	for i, in := range input {
		fragments[i] = makeRawFragment(in)
	}
	return fragments
}

func makeRawFragment(term any, params ...any) RawQuery {
	if s, ok := term.(string); ok {
		return &rawQuery{term: s, params: params}
	}
	if len(params) != 0 {
		panic(fmt.Errorf("term can only take params if it is a string"))
	}
	if q, ok := term.(*rawQuery); ok {
		return q
	} else if q, ok := term.(SelectQuery); ok {
		return &rawQuery{
			term:   "?",
			params: []any{q},
		}
	} else {
		panic("unsupported type")
	}
}

func Raw(sql string, params ...any) RawQuery {
	return makeRawFragment(sql, params...)
}

func andor(joiner string, terms ...any) RawQuery {
	if len(terms) == 0 {
		panic("no terms given")
	}
	if len(terms) == 1 {
		q := makeRawFragment(terms[0])
		return q
	}
	parts := make([]string, len(terms))
	params := make([]any, len(terms))
	for i, term := range terms {
		parts[i] = "(?)"
		if s, ok := term.(string); ok {
			params[i] = makeRawFragment(s)
		} else {
			params[i] = term
		}
	}
	return &rawQuery{
		term:   strings.Join(parts, joiner),
		params: params,
	}
}

func And(terms ...any) RawQuery {
	return andor(" AND ", terms...)
}
func Or(terms ...any) RawQuery {
	return andor(" OR ", terms...)
}
func Not(term any) RawQuery {
	var param any
	if s, ok := term.(string); ok {
		param = makeRawFragment(s)
	} else {
		param = term
	}
	return &rawQuery{
		term:   "NOT(?)",
		params: []any{param},
	}
}

func Exists(subquery SelectQuery) RawQuery {
	return &rawQuery{
		term:   "EXISTS ?",
		params: []any{subquery},
	}
}

func InsertInto(table string, columns ...string) InsertQuery {
	return &insertQuery{
		table:   table,
		columns: columns,
	}
}

type InsertQuery interface {
	Query

	Values(vals ...any) InsertQuery
	Query(query Query) InsertQuery
	Returning(exprs ...any) InsertQuery
}

type insertQuery struct {
	table     string
	columns   []string
	values    [][]RawQuery
	query     Query
	returning []RawQuery
}

func (q *insertQuery) build(sql *strings.Builder, p *params) {
	sql.WriteString("INSERT INTO " + q.table)
	if len(q.columns) > 0 {
		sql.WriteString(" (")
		sql.WriteString(strings.Join(q.columns, ", "))
		sql.WriteString(")")
	}

	if q.query == nil {
		sql.WriteString(" VALUES ")
		for idx, row := range q.values {
			sql.WriteRune('(')
			buildFragments(sql, p, row, ", ")
			sql.WriteRune(')')
			if idx < len(q.values)-1 {
				sql.WriteString(", ")
			}
		}
	} else {
		sql.WriteRune(' ')
		q.query.build(sql, p)
	}

	if len(q.returning) > 0 {
		sql.WriteString(" RETURNING ")
		buildFragments(sql, p, q.returning, ", ")
	}

}

func (q *insertQuery) Build() SqlQuery {
	var p params
	var sql strings.Builder
	q.build(&sql, &p)
	return SqlQuery{
		Query:      sql.String(),
		Parameters: p,
	}
}

func (q *insertQuery) Values(vals ...any) InsertQuery {
	if q.query != nil {
		panic("cannot use both Query and Values on InsertQuery")
	}
	fragments := make([]RawQuery, len(vals))
	for idx, val := range vals {
		if fragment, ok := val.(*rawQuery); ok {
			fragments[idx] = fragment
		} else {
			fragments[idx] = makeRawFragment("?", val)
		}
	}
	q.values = append(q.values, fragments)
	return q
}

func (q *insertQuery) Query(query Query) InsertQuery {
	if len(q.values) > 0 {
		panic("cannot use both Query and Values on InsertQuery")
	}
	q.query = query
	return q
}

func (q *insertQuery) Returning(exprs ...any) InsertQuery {
	q.returning = append(q.returning, makeRawFragments(exprs)...)
	return q
}

func Update(table string) UpdateQuery {
	return &updateQuery{
		table: table,
	}
}

type UpdateQuery interface {
	Query

	Set(col string, value any) UpdateQuery
	From(table any) UpdateQuery
	Where(condition any, params ...any) UpdateQuery
	Returning(exprs ...any) UpdateQuery
}

type updateQuery struct {
	table     string
	cols      []h.T2[string, RawQuery]
	where     []RawQuery
	from      []RawQuery
	returning []RawQuery
}

func (q *updateQuery) From(table any) UpdateQuery {
	q.from = append(q.from, makeRawFragment(table))
	return q
}

func (q *updateQuery) Where(condition any, params ...any) UpdateQuery {
	q.where = append(q.where, makeRawFragment(condition, params...))
	return q
}

func (q *updateQuery) Returning(exprs ...any) UpdateQuery {
	q.returning = append(q.returning, makeRawFragments(exprs)...)
	return q
}

func (q *updateQuery) Build() SqlQuery {
	var p params
	var sql strings.Builder
	q.build(&sql, &p)
	return SqlQuery{
		Query:      sql.String(),
		Parameters: p,
	}
}

func (q *updateQuery) build(sql *strings.Builder, p *params) {
	sql.WriteString("UPDATE ")
	sql.WriteString(q.table)
	sql.WriteString(" SET ")
	for i, col := range q.cols {
		sql.WriteString(col.First)
		sql.WriteString(" = ")
		col.Second.build(sql, p)
		if i < len(q.cols)-1 {
			sql.WriteString(", ")
		}
	}

	if len(q.from) > 0 {
		sql.WriteString(" FROM ")
		buildFragments(sql, p, q.from, ", ")
	}

	if len(q.where) > 0 {
		wrapConds := len(q.where) > 1
		for idx, where := range q.where {
			if idx == 0 {
				sql.WriteString(" WHERE ")
			} else {
				sql.WriteString(" AND ")
			}
			if wrapConds {
				sql.WriteString("(")
			}
			where.build(sql, p)
			if wrapConds {
				sql.WriteString(")")
			}
		}
	}

	if len(q.returning) > 0 {
		sql.WriteString(" RETURNING ")
		buildFragments(sql, p, q.returning, ", ")
	}
}

func (q *updateQuery) Set(col string, value any) UpdateQuery {
	var fragment RawQuery
	if f, ok := value.(*rawQuery); ok {
		fragment = f
	} else {
		fragment = makeRawFragment("?", value)
	}
	q.cols = append(q.cols, h.NewT2(col, fragment))
	return q
}

func DeleteFrom(table string) DeleteQuery {
	return &deleteQuery{
		table: table,
	}
}

type DeleteQuery interface {
	Query

	Using(table any) DeleteQuery
	Where(condition any, params ...any) DeleteQuery
	Returning(exprs ...any) DeleteQuery
}

type deleteQuery struct {
	table     string
	cols      []h.T2[string, RawQuery]
	where     []RawQuery
	from      []RawQuery
	returning []RawQuery
}

func (q *deleteQuery) Build() SqlQuery {
	//TODO implement me
	panic("implement me")
}

func (q *deleteQuery) build(sql *strings.Builder, p *params) {
	//TODO implement me
	panic("implement me")
}

func (q *deleteQuery) Using(table any) DeleteQuery {
	//TODO implement me
	panic("implement me")
}

func (q *deleteQuery) Where(condition any, params ...any) DeleteQuery {
	//TODO implement me
	panic("implement me")
}

func (q *deleteQuery) Returning(exprs ...any) DeleteQuery {
	//TODO implement me
	panic("implement me")
}
