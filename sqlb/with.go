package sqlb

import "strings"

func With(name string, query any, params ...any) WithQuery {
	if s, ok := query.(string); ok {
		query = "(" + s + ")"
	}
	return &withQuery{
		withs: []RawQuery{makeRawFragment(name+" AS ?", makeRawFragment(query, params...))},
	}
}

type WithQuery interface {
	With(name string, query any, params ...any) WithQuery
	Select(args ...any) SelectQuery
	InsertInto(table string, columns ...string) InsertQuery
	Update(table string) UpdateQuery
	DeleteFrom(table string) DeleteQuery
	Raw(query string, params ...any) RawQuery
}

type withQuery struct {
	withs []RawQuery
}

func (q *withQuery) With(name string, query any, params ...any) WithQuery {
	if s, ok := query.(string); ok {
		query = "(" + s + ")"
	}
	q.withs = append(q.withs, makeRawFragment(name+" AS ?", makeRawFragment(query, params...)))
	return q
}

func (q *withQuery) Select(cols ...any) SelectQuery {
	return &selectQuery{
		with:    q.withs,
		columns: makeRawFragments(cols),
	}
}

func (q *withQuery) InsertInto(table string, columns ...string) InsertQuery {
	return &insertQuery{
		with:    q.withs,
		table:   table,
		columns: columns,
	}
}

func (q *withQuery) Update(table string) UpdateQuery {
	return &updateQuery{
		with:  q.withs,
		table: table,
	}
}

func (q *withQuery) DeleteFrom(table string) DeleteQuery {
	return &deleteQuery{
		with:  q.withs,
		table: table,
	}
}

func (q *withQuery) Raw(query string, params ...any) RawQuery {
	allParams := make([]any, len(q.withs), len(q.withs)+len(params))
	for i, w := range q.withs {
		allParams[i] = w
	}
	allParams = append(allParams, params...)

	return makeRawFragment("WITH"+strings.Repeat(" ?", len(q.withs))+" "+query, allParams...)
}
