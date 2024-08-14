package sqlb

type SqlQuery struct {
	Query      string
	Parameters []any
}

type Query interface{}

type SelectQuery interface {
	From(table any) SelectQuery
	Where(condition any, params ...any) SelectQuery
	Limit(limit int) SelectQuery
	Offset(offset int) SelectQuery
	OrderBy(fields ...string) SelectQuery
	Join(table any, on any) SelectQuery
	InnerJoin(table any, on any, params ...any) SelectQuery
	LeftJoin(table any, on any, params ...any) SelectQuery
	RightJoin(table any, on any, params ...any) SelectQuery
	FullJoin(table any, on any, params ...any) SelectQuery
	CrossJoin(table any, on any, params ...any) SelectQuery

	Build() SqlQuery
}

type WithQuery interface {
	With(name string, query Query) WithQuery
	Select(args ...any) SelectQuery
}

type QueryAs interface {
}

func As(query any, name string) QueryAs {

}

func With(name string, query Query) WithQuery {

}

func Select(args ...any) SelectQuery {

}

type Term interface {
}

func Predicate(condition any, params ...any) Term {}

func And(terms ...Term) Term {}
func Or(terms ...Term) Term  {}
func Not(term Term) Term     {}
