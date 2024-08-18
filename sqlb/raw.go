package sqlb

import (
	"github.com/DataDog/go-sqllexer"
	"strings"
)

func Raw(sql string, params ...any) RawQuery {
	return makeRawFragment(sql, params...)
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
		Sql:        sql.String(),
		Parameters: p,
	}
}
