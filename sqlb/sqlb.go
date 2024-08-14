package sqlb

import (
	"fmt"
	"github.com/DataDog/go-sqllexer"
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
	From(table any) SelectQuery
	Where(condition any, params ...any) SelectQuery
	Limit(limit int) SelectQuery
	Offset(offset int) SelectQuery
	OrderBy(fields ...string) SelectQuery
	Join(table any, on any, params ...any) SelectQuery
	InnerJoin(table any, on any, params ...any) SelectQuery
	LeftJoin(table any, on any, params ...any) SelectQuery
	RightJoin(table any, on any, params ...any) SelectQuery
	FullJoin(table any, on any, params ...any) SelectQuery
	CrossJoin(table any) SelectQuery

	Build() SqlQuery
}

type selectQuery struct {
	columns []any
	from    []any
	joins   []selectJoin
	where   []TermQueryFragment
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
	condition TermQueryFragment
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

func (s *selectQuery) Limit(limit int) SelectQuery {
	//TODO implement me
	panic("implement me")
}

func (s *selectQuery) Offset(offset int) SelectQuery {
	//TODO implement me
	panic("implement me")
}

func (s *selectQuery) OrderBy(fields ...string) SelectQuery {
	//TODO implement me
	panic("implement me")
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

func (s *selectQuery) buildSelect(sql *strings.Builder, p *params) {
	var colStrings []string
	var buildSelectCol func(col any) string
	buildSelectCol = func(col any) string {
		if as, ok := col.(queryAs); ok {
			str := buildSelectCol(as.query)
			return fmt.Sprintf("%s AS %s", str, as.name)
		} else if s, ok := col.(string); ok {
			return s
		} else if q, ok := col.(TermQueryFragment); ok {
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

	cols := strings.Join(colStrings, ", ")
	sql.WriteString(fmt.Sprintf("SELECT %s", cols))
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
	for idx, where := range s.where {
		if idx == 0 {
			sql.WriteString(" WHERE ")
		} else {
			sql.WriteString(" AND ")
		}
		sql.WriteString(where.build(p))
	}
}

type params []any

func (p *params) append(param any) string {
	*p = append(*p, param)
	return "$" + strconv.Itoa(len(*p))
}

func (s *selectQuery) build(p *params) string {
	var sql strings.Builder
	s.buildSelect(&sql, p)
	s.buildFrom(&sql, p)
	s.buildJoin(&sql, p)
	s.buildWhere(&sql, p)
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
	With(name string, query Query) WithQuery
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

func With(name string, query Query) WithQuery {
	panic("not implemented")

}

func Select(col any, cols ...any) SelectQuery {
	c := make([]any, 0, len(cols)+1)
	c = append(c, col)
	c = append(c, cols...)
	return &selectQuery{columns: c}

}

type TermQueryFragment struct {
	term   any
	params []any
}

func (t *TermQueryFragment) build(p *params) string {
	if s, ok := t.term.(string); ok {
		lexer := sqllexer.New(s)
		tokens := lexer.ScanAll()
		var parts []string
		paramIdx := 0
		for _, token := range tokens {
			if token.Type == sqllexer.OPERATOR && token.Value == "?" {
				param := t.params[paramIdx]
				if q, ok := param.(*selectQuery); ok {
					parts = append(parts, q.build(p))
				} else if q, ok := param.(*TermQueryFragment); ok {
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
	} else {
		panic("not implemented")
	}
}

func Term(term any, params ...any) TermQueryFragment {
	return TermQueryFragment{term: term, params: params}
}

func And(terms ...TermQueryFragment) TermQueryFragment {
	panic("not implemented")
}
func Or(terms ...TermQueryFragment) TermQueryFragment {
	panic("not implemented")
}
func Not(term TermQueryFragment) TermQueryFragment {
	panic("not implemented")
}

func Exists(subquery SelectQuery) TermQueryFragment {
	return TermQueryFragment{
		term:   "EXISTS(?)",
		params: []any{subquery},
	}
}

func (s *SqlQuery) concat(other *SqlQuery) SqlQuery {
	params := make([]any, 0, len(s.Parameters)+len(other.Parameters))
	params = append(params, s.Parameters...)
	params = append(params, other.Parameters...)

	return SqlQuery{
		Query:      fmt.Sprintf("%s %s", s.Query, other.Query),
		Parameters: params,
	}
}
