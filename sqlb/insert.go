package sqlb

import (
	"fmt"
	"strings"
)

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
	OnConflict() InsertQueryConflict
	OnConflictRaw(raw string, params ...any) InsertQuery
	Returning(exprs ...any) InsertQuery
}

type InsertQueryConflict interface {
	Query

	Values(vals ...any) InsertQuery
	Query(query Query) InsertQuery
	Returning(exprs ...any) InsertQuery

	DoNothing() InsertQueryConflict
	DoUpdate() InsertQueryConflictUpdate
	Cols(cols ...string) InsertQueryConflict
	Constraint(name string) InsertQueryConflict
}

type InsertQueryConflictUpdate interface {
	Query

	Values(vals ...any) InsertQuery
	Query(query Query) InsertQuery
	Returning(exprs ...any) InsertQuery

	Cols(cols ...string) InsertQueryConflict
	Constraint(name string) InsertQueryConflict

	Set(name string, value any, params ...any) InsertQueryConflictUpdate
	Where(condition any, params ...any) InsertQueryConflictUpdate
}

type insertQuery struct {
	with          []RawQuery
	table         string
	columns       []string
	values        [][]RawQuery
	query         Query
	returning     []RawQuery
	onConflict    *insertQueryConflict
	onConflictRaw RawQuery
}

type insertQueryConflict struct {
	constraint string
	cols       []string
	set        []RawQuery
	where      []RawQuery
	strategy   insertQueryConflictStrategy
}

type insertQueryConflictStrategy int

const (
	insertConflictInvalid insertQueryConflictStrategy = iota
	insertConflictDoNothing
	insertConflictDoUpdate
)

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

	if q.onConflict != nil {
		if q.onConflict.strategy == insertConflictInvalid {
			panic(fmt.Errorf("called OnConflict() but neither DoNothing() nor DoUpdate()"))
		}
		sql.WriteString(" ON CONFLICT ")

		if len(q.onConflict.cols) > 0 {
			sql.WriteRune('(')
			sql.WriteString(strings.Join(q.onConflict.cols, ", "))
			sql.WriteString(") ")
		} else if q.onConflict.constraint != "" {
			sql.WriteString("ON CONSTRAINT ")
			sql.WriteString(q.onConflict.constraint)
			sql.WriteRune(' ')
		}

		if q.onConflict.strategy == insertConflictDoNothing {
			sql.WriteString("DO NOTHING")
		} else {
			sql.WriteString("DO UPDATE SET ")
			buildFragments(sql, p, q.onConflict.set, ", ")
			if len(q.onConflict.where) > 0 {
				buildWhere(sql, p, q.onConflict.where)
			}
		}
	} else if q.onConflictRaw != nil {
		sql.WriteRune(' ')
		q.onConflictRaw.build(sql, p)
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
		Sql:        sql.String(),
		Parameters: p,
	}
}

func (q *insertQuery) Values(vals ...any) InsertQuery {
	if q.query != nil {
		panic(fmt.Errorf("cannot use both Query and Values on InsertQuery"))
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
		panic(fmt.Errorf("cannot use both Query and Values on InsertQuery"))
	}
	q.query = query
	return q
}

func (q *insertQuery) OnConflictRaw(raw string, params ...any) InsertQuery {
	if q.onConflict != nil {
		panic(fmt.Errorf("cannot use both OnConflict and OnConflictRaw"))
	}
	q.onConflictRaw = makeRawFragment(raw, params...)
	return q
}

func (q *insertQuery) OnConflict() InsertQueryConflict {
	if q.onConflictRaw != nil {
		panic(fmt.Errorf("cannot use both OnConflict and OnConflictRaw"))
	}
	if q.onConflict != nil {
		panic(fmt.Errorf("cannot call OnConflict more than once"))
	}
	q.onConflict = &insertQueryConflict{}
	return q
}

func (q *insertQuery) Cols(cols ...string) InsertQueryConflict {
	q.onConflict.cols = cols
	return q
}

func (q *insertQuery) Constraint(name string) InsertQueryConflict {
	q.onConflict.constraint = name
	return q
}

func (q *insertQuery) DoNothing() InsertQueryConflict {
	q.onConflict.strategy = insertConflictDoNothing
	return q
}

func (q *insertQuery) DoUpdate() InsertQueryConflictUpdate {
	q.onConflict.strategy = insertConflictDoUpdate
	return q
}

func (q *insertQuery) Set(name string, value any, params ...any) InsertQueryConflictUpdate {
	if q.onConflict.strategy != insertConflictDoUpdate {
		panic(fmt.Errorf("cannot call Set() without DoUpdate()"))
	}
	q.onConflict.set = append(q.onConflict.set, makeRawFragment(name+" = ?", makeRawFragment(value, params...)))
	return q
}

func (q *insertQuery) Where(condition any, params ...any) InsertQueryConflictUpdate {
	if q.onConflict.strategy != insertConflictDoUpdate {
		panic(fmt.Errorf("cannot call Where() without DoUpdate()"))
	}
	q.onConflict.where = append(q.onConflict.where, makeRawFragment(condition, params...))
	return q
}

func (q *insertQuery) Returning(exprs ...any) InsertQuery {
	q.returning = append(q.returning, makeRawFragments(exprs)...)
	return q
}
