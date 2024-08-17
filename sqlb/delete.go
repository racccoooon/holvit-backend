package sqlb

import "strings"

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
	with      []RawQuery
	table     string
	using     []RawQuery
	where     []RawQuery
	returning []RawQuery
}

func (q *deleteQuery) Build() SqlQuery {
	var p params
	var sql strings.Builder
	q.build(&sql, &p)
	return SqlQuery{
		Query:      sql.String(),
		Parameters: p,
	}
}

func (q *deleteQuery) build(sql *strings.Builder, p *params) {
	buildWith(sql, p, q.with)
	sql.WriteString("DELETE FROM ")
	sql.WriteString(q.table)

	if len(q.using) > 0 {
		sql.WriteString(" USING ")
		buildFragments(sql, p, q.using, ", ")
	}

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

	if len(q.returning) > 0 {
		sql.WriteString(" RETURNING ")
		buildFragments(sql, p, q.returning, ", ")
	}
}

func (q *deleteQuery) Using(table any) DeleteQuery {
	q.using = append(q.using, makeRawFragment(table))
	return q
}

func (q *deleteQuery) Where(condition any, params ...any) DeleteQuery {
	q.where = append(q.where, makeRawFragment(condition, params...))
	return q
}

func (q *deleteQuery) Returning(exprs ...any) DeleteQuery {
	q.returning = append(q.returning, makeRawFragments(exprs)...)
	return q
}
