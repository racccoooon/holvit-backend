package sqlb

import "strings"

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
	with      []RawQuery
	table     string
	columns   []string
	values    [][]RawQuery
	query     Query
	returning []RawQuery
}

func (q *insertQuery) build(sql *strings.Builder, p *params) {
	buildWith(sql, p, q.with)
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
