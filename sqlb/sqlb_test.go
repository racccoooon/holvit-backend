package sqlb

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_SelectConst(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		query := Select("true").Build()
		assert.Equal(t, "SELECT true", query.Query)
		assert.Empty(t, query.Parameters)
	})
	t.Run("false", func(t *testing.T) {
		query := Select("false").Build()
		assert.Equal(t, "SELECT false", query.Query)
		assert.Empty(t, query.Parameters)
	})
	t.Run("quoted string", func(t *testing.T) {
		query := Select(`"123"`).Build()
		assert.Equal(t, `SELECT "123"`, query.Query)
		assert.Empty(t, query.Parameters)
	})
}

func Test_SelectConstMultiple(t *testing.T) {
	t.Run("two", func(t *testing.T) {
		query := Select("true", "false").Build()
		assert.Equal(t, "SELECT true, false", query.Query)
		assert.Empty(t, query.Parameters)
	})
}

func Test_SelectTermParam(t *testing.T) {
	query := Select(Term("?", true)).Build()
	assert.Equal(t, "SELECT $1", query.Query)
	assert.Equal(t, []any{true}, query.Parameters)
}

func Test_SelectTermMultipleParams(t *testing.T) {
	query := Select(Term("? + ?", 1, 2)).Build()
	assert.Equal(t, "SELECT $1 + $2", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
}

func Test_SelectTerm(t *testing.T) {
	query := Select(Term("true")).Build()
	assert.Equal(t, "SELECT true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectMultipleTermsWithParams(t *testing.T) {
	query := Select(Term("? + ?", 1, 2), Term("?", true)).Build()
	assert.Equal(t, "SELECT $1 + $2, $3", query.Query)
	assert.Equal(t, []any{1, 2, true}, query.Parameters)
}

func Test_SelectTermWithQuotedStringAndParam(t *testing.T) {
	query := Select(Term(`"foo?" + ?`, "bar")).Build()
	assert.Equal(t, `SELECT "foo?" + $1`, query.Query)
	assert.Equal(t, []any{"bar"}, query.Parameters)
}

func Test_SelectAs(t *testing.T) {
	query := Select(As(Term("? + ?", 1, 2), "bar"), As("foo", "foo2"), "a as b").Build()
	assert.Equal(t, "SELECT $1 + $2 AS bar, foo AS foo2, a as b", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
}

func Test_Subselect(t *testing.T) {
	query := Select(Term("?", 1), As(Select(Term("?", 2)), "b"), Term("?", 3)).Build()
	assert.Equal(t, "SELECT $1, (SELECT $2) AS b, $3", query.Query)
	assert.Equal(t, []any{1, 2, 3}, query.Parameters)
}

func Test_SelectFrom(t *testing.T) {
	query := Select("foo", "bar").From("foobar").Build()
	assert.Equal(t, "SELECT foo, bar FROM foobar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFromMultiple(t *testing.T) {
	query := Select("foo", "bar").From("foobar").From("foobaz").Build()
	assert.Equal(t, "SELECT foo, bar FROM foobar, foobaz", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFromSubquery(t *testing.T) {
	query := Select("foo", "bar").From(Select("a as foo", "b as bar").From("ab")).Build()
	assert.Equal(t, "SELECT foo, bar FROM (SELECT a as foo, b as bar FROM ab)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFromSubqueryWithParams(t *testing.T) {
	query := Select("foo", "bar", Term("?", 3)).From(Select("1 as foo", As(Term("?", 2), "bar"))).Build()
	assert.Equal(t, "SELECT foo, bar, $1 FROM (SELECT 1 as foo, $2 AS bar)", query.Query)
	assert.Equal(t, []any{3, 2}, query.Parameters)
}

func Test_SelectFromJoin(t *testing.T) {
	query := Select("*").From("foo").Join("bar", "foo.id = bar.foo_id").Build()
	assert.Equal(t, "SELECT * FROM foo JOIN bar ON foo.id = bar.foo_id", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFromMultipleJoinsParams(t *testing.T) {
	query := Select("*", Term("?", 2)).From("foo").Join("bar", "foo.id = bar.foo_id and foo.x = ?", 3).Join("baz", "bar.id = baz.bar_id").Build()
	assert.Equal(t, "SELECT *, $1 FROM foo JOIN bar ON foo.id = bar.foo_id and foo.x = $2 JOIN baz ON bar.id = baz.bar_id", query.Query)
	assert.Equal(t, []any{2, 3}, query.Parameters)
}

func Test_SelectInnerJoin(t *testing.T) {
	query := Select("*").From("foo").InnerJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo INNER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectLeftJoin(t *testing.T) {
	query := Select("*").From("foo").LeftJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo LEFT OUTER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectRightJoin(t *testing.T) {
	query := Select("*").From("foo").RightJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo RIGHT OUTER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFullJoin(t *testing.T) {
	query := Select("*").From("foo").FullJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo FULL OUTER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectCrossJoin(t *testing.T) {
	query := Select("*").From("foo").CrossJoin("bar").Build()
	assert.Equal(t, "SELECT * FROM foo CROSS JOIN bar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectExists(t *testing.T) {
	query := Select(Exists(Select("1").From("foo"))).Build()
	assert.Equal(t, "SELECT EXISTS(SELECT 1 FROM foo)", query.Query)
	assert.Empty(t, query.Parameters)
}

func asdf(t *testing.T) {
	// select foo, bar from something where x > 1 and y < x limit 10 offset 3

	Select("foo", "bar").
		From("something").
		Where("x > ?", 1).
		Where("y IS NULL").
		Limit(10).
		Offset(3)

	// select count(*) over (), foo, bar from something
	Select("count(*) over ()", "foo", "bar").From("something")

	// select (select count(*) from bar where foo.a == foo.b) as x from foo where asdf == 0
	Select(
		As(Select("count(*)").From("bar").Where("foo.a == foo.b"), "x"), // as X how?
	).From("foo").Where("asdf == ?", 0)

	cond1 := Term("foo.x == ?", 7)
	cond2 := Term("foo.y any ?", []int{1, 2, 3})
	cond3 := Term("foo.z > foo.asdf")

	cond := And(cond1, cond2)
	if true {
		cond = Or(cond, cond3)
	}

	Select("...").Where(cond)

	Select("...").Where("x in ?", Select("id").From("foos"))

	// select foo.id, foo.name, bar.something from foo join bar on foo.bar_id == bar.id and bar.asdf is null

	Select("foo.id", "foo.name", "bar.something").From("foo").Join("bar", "foo.bar_id == bar.id and bar.asdf is null and")

	// WITH names AS (SELECT name FROM people) SELECT x, y FROM asdf WHERE asdf.bar IN names

	With("names", Select("name").From("people")).Select("x", "y").From("asdf").Where("asdf.bar IN names")

}
