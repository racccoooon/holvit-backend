package sqlb

import (
	"fmt"
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

type params []any

func (p *params) append(param any) string {
	*p = append(*p, param)
	return "$" + strconv.Itoa(len(*p))
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
		panic(fmt.Errorf("term can only take params if term is a string"))
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

func buildFragments(sql *strings.Builder, p *params, fragments []RawQuery, joiner string) {
	for i, fragment := range fragments {
		fragment.build(sql, p)
		if i < len(fragments)-1 {
			sql.WriteString(joiner)
		}
	}
}

func buildWith(sql *strings.Builder, p *params, withs []RawQuery) {
	if len(withs) == 0 {
		return
	}
	sql.WriteString("WITH ")
	buildFragments(sql, p, withs, ", ")
	sql.WriteRune(' ')
}
