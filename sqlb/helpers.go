package sqlb

import "strings"

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

func As(query any, name string) RawQuery {
	return &rawQuery{
		term:   "? AS " + name,
		params: []any{makeRawFragment(query)},
	}
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
