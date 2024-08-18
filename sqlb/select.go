package sqlb

import (
	"strings"
)

func Select(col any, cols ...any) SelectQuery {
	c := make([]any, 0, len(cols)+1)
	c = append(c, col)
	c = append(c, cols...)
	return &selectQuery{columns: makeRawFragments(c)}

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
	RawJoin(join any, params ...any) SelectQuery
	Where(condition any, params ...any) SelectQuery
	GroupBy(groupingElements ...any) SelectQuery
	Having(condition any, params ...any) SelectQuery
	OrderBy(fields ...any) SelectQuery
	Limit(limit any) SelectQuery
	Offset(offset any) SelectQuery
	LockForUpdate(skipLocked bool) SelectQuery
}

type selectQuery struct {
	with          []RawQuery
	columns       []RawQuery
	from          []RawQuery
	joins         []RawQuery
	where         []RawQuery
	groupBy       []RawQuery
	having        []RawQuery
	distinct      bool
	distinctOn    []RawQuery
	orderBy       []RawQuery
	limit         RawQuery
	offset        RawQuery
	lockForUpdate bool
	skipLocked    bool
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
		s.limit = makeRawFragment("?", num)
	} else {
		s.limit = makeRawFragment(limit)
	}
	return s
}

func (s *selectQuery) Offset(offset any) SelectQuery {
	if num, ok := offset.(int); ok {
		s.offset = makeRawFragment("?", num)
	} else {
		s.offset = makeRawFragment(offset)
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

func (s *selectQuery) join(join string, table any, on any, params []any) SelectQuery {
	s.joins = append(s.joins, makeRawFragment(join+" ? ON ?", makeRawFragment(table), makeRawFragment(on, params...)))
	return s
}

func (s *selectQuery) joinAs(join string, table any, as string, on any, params []any) SelectQuery {
	s.joins = append(s.joins, makeRawFragment(join+" ? AS "+as+" ON ?", makeRawFragment(table), makeRawFragment(on, params...)))
	return s
}

func (s *selectQuery) Join(table any, on any, params ...any) SelectQuery {
	s.join("JOIN", table, on, params)
	return s
}

func (s *selectQuery) JoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joinAs("JOIN", table, name, on, params)
	return s
}

func (s *selectQuery) InnerJoin(table any, on any, params ...any) SelectQuery {
	s.join("INNER JOIN", table, on, params)
	return s
}

func (s *selectQuery) InnerJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joinAs("INNER JOIN", table, name, on, params)
	return s
}

func (s *selectQuery) LeftJoin(table any, on any, params ...any) SelectQuery {
	s.join("LEFT OUTER JOIN", table, on, params)
	return s
}

func (s *selectQuery) LeftJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joinAs("LEFT OUTER JOIN", table, name, on, params)
	return s
}

func (s *selectQuery) RightJoin(table any, on any, params ...any) SelectQuery {
	s.join("RIGHT OUTER JOIN", table, on, params)
	return s
}

func (s *selectQuery) RightJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joinAs("RIGHT OUTER JOIN", table, name, on, params)
	return s
}

func (s *selectQuery) FullJoin(table any, on any, params ...any) SelectQuery {
	s.join("FULL OUTER JOIN", table, on, params)
	return s
}

func (s *selectQuery) FullJoinAs(table any, name string, on any, params ...any) SelectQuery {
	s.joinAs("FULL OUTER JOIN", table, name, on, params)
	return s
}

func (s *selectQuery) CrossJoin(table any) SelectQuery {
	s.joins = append(s.joins, makeRawFragment("CROSS JOIN ?", makeRawFragment(table)))
	return s
}

func (s *selectQuery) CrossJoinAs(table any, name string) SelectQuery {
	s.joins = append(s.joins, makeRawFragment("CROSS JOIN ? AS "+name, makeRawFragment(table)))
	return s
}

func (s *selectQuery) RawJoin(join any, params ...any) SelectQuery {
	s.joins = append(s.joins, makeRawFragment(join, params...))
	return s
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
	if len(s.joins) > 0 {
		sql.WriteRune(' ')
		buildFragments(sql, p, s.joins, " ")
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
	if s.limit != nil {
		sql.WriteString(" LIMIT ")
		s.limit.build(sql, p)
	}
	if s.offset != nil {
		sql.WriteString(" OFFSET ")
		s.offset.build(sql, p)
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

func (s *selectQuery) build(sql *strings.Builder, p *params) {
	buildWith(sql, p, s.with)
	s.buildSelect(sql, p)
	s.buildFrom(sql, p)
	s.buildJoin(sql, p)
	buildWhere(sql, p, s.where)
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
		Sql:        sql.String(),
		Parameters: p,
	}
}
