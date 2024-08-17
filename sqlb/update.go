package sqlb

import (
	"strings"
)

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

type updateCol struct {
	col   string
	value RawQuery
}

type updateQuery struct {
	with      []RawQuery
	table     string
	cols      []updateCol
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
	buildWith(sql, p, q.with)
	sql.WriteString("UPDATE ")
	sql.WriteString(q.table)
	sql.WriteString(" SET ")
	for i, col := range q.cols {
		sql.WriteString(col.col)
		sql.WriteString(" = ")
		col.value.build(sql, p)
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
	q.cols = append(q.cols, updateCol{col, fragment})
	return q
}
