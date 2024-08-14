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

func Test_SelectMultiple(t *testing.T) {
	query := Select("foo").Select("bar").Build()
	assert.Equal(t, "SELECT foo, bar", query.Query)
	assert.Empty(t, query.Parameters)
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
	assert.Equal(t, "SELECT EXISTS (SELECT 1 FROM foo)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectWhere(t *testing.T) {
	query := Select("*").From("foo").Where("true").Build()
	assert.Equal(t, "SELECT * FROM foo WHERE true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectWhereParams(t *testing.T) {
	query := Select("foo", Term("?", 1)).From("bar").Where("x > ?", 2).Build()
	assert.Equal(t, "SELECT foo, $1 FROM bar WHERE x > $2", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
}

func Test_SelectWhereTerm(t *testing.T) {
	query := Select("*").From("foo").Where(And("true and false", "true")).Build()
	assert.Equal(t, "SELECT * FROM foo WHERE (true and false) AND (true)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectWhereTermComplex(t *testing.T) {
	query := Select("*").From("foo").Where(Or(
		And(
			Term("x = ?", 1),
			"asdf < 10",
		),
		And(
			Term("bar any ?", []int{2, 3, 4}),
			Not("z IS NULL"),
		),
	)).Build()
	assert.Equal(t, "SELECT * FROM foo WHERE ((x = $1) AND (asdf < 10)) OR ((bar any $2) AND (NOT(z IS NULL)))", query.Query)
	assert.Equal(t, []any{1, []int{2, 3, 4}}, query.Parameters)
}

func Test_SelectWhereMultiple(t *testing.T) {
	query := Select("*").From("foo").Where("a > 1").Where("b < 1").Build()
	assert.Equal(t, "SELECT * FROM foo WHERE (a > 1) AND (b < 1)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectDistinct(t *testing.T) {
	query := Select("*").From("foo").Distinct().Build()
	assert.Equal(t, "SELECT DISTINCT * FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectDistinctOn(t *testing.T) {
	query := Select("*").From("foo").Distinct("a", "b").Build()
	assert.Equal(t, "SELECT DISTINCT ON (a, b) * FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectDistinctOnMultiple(t *testing.T) {
	query := Select("*").From("foo").Distinct("a", "b").Distinct("c").Build()
	assert.Equal(t, "SELECT DISTINCT ON (a, b, c) * FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectDistinctOnComplex(t *testing.T) {
	query := Select("*").From("foo").Distinct(Term("(a / ?)::int", 37)).Build()
	assert.Equal(t, "SELECT DISTINCT ON ((a / $1)::int) * FROM foo", query.Query)
	assert.Equal(t, []any{37}, query.Parameters)
}

func Test_SelectOrderBy(t *testing.T) {
	query := Select("*").From("foo").OrderBy("name desc", "id asc").Build()
	assert.Equal(t, "SELECT * FROM foo ORDER BY name desc, id asc", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectOrderByMultiple(t *testing.T) {
	query := Select("*").From("foo").OrderBy("name desc", "foo").OrderBy("id asc").Build()
	assert.Equal(t, "SELECT * FROM foo ORDER BY name desc, foo, id asc", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectOrderByComplex(t *testing.T) {
	query := Select("*").From("foobar").OrderBy(Term("CASE WHEN name = ? THEN foo ELSE ? END", "foobar", 7)).Build()
	assert.Equal(t, "SELECT * FROM foobar ORDER BY CASE WHEN name = $1 THEN foo ELSE $2 END", query.Query)
	assert.Equal(t, []any{"foobar", 7}, query.Parameters)
}

func Test_SelectLimit(t *testing.T) {
	query := Select("*").From("foobar").Limit(20).Build()
	assert.Equal(t, "SELECT * FROM foobar LIMIT $1", query.Query)
	assert.Equal(t, []any{20}, query.Parameters)
}

func Test_SelectLimitImplicitTerm(t *testing.T) {
	query := Select("*").From("foobar").Limit("foo").Build()
	assert.Equal(t, "SELECT * FROM foobar LIMIT foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectLimitExplicitTerm(t *testing.T) {
	query := Select("*").From("foobar").Limit(Term("? + 2", 10)).Build()
	assert.Equal(t, "SELECT * FROM foobar LIMIT $1 + 2", query.Query)
	assert.Equal(t, []any{10}, query.Parameters)
}

func Test_SelectLimitSubquery(t *testing.T) {
	query := Select("*").From("foo").Limit(Select("count(*)").From("bar")).Build()
	assert.Equal(t, "SELECT * FROM foo LIMIT (SELECT count(*) FROM bar)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectForUpdate(t *testing.T) {
	query := Select("*").From("foo").LockForUpdate(false).Build()
	assert.Equal(t, "SELECT * FROM foo FOR UPDATE", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectForUpdateSkipLocked(t *testing.T) {
	query := Select("*").From("foo").LockForUpdate(true).Build()
	assert.Equal(t, "SELECT * FROM foo FOR UPDATE SKIP LOCKED", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_WithSelectString(t *testing.T) {
	query := With("foo", "SELECT id, name FROM bar").Select("name").From("foo").Build()
	assert.Equal(t, "WITH foo AS (SELECT id, name FROM bar) SELECT name FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_WithSelectStringParams(t *testing.T) {
	query := With("foo", "SELECT id, name FROM bar WHERE x > ?", 21).Select("name").From("foo").Build()
	assert.Equal(t, "WITH foo AS (SELECT id, name FROM bar WHERE x > $1) SELECT name FROM foo", query.Query)
	assert.Equal(t, []any{21}, query.Parameters)
}

func Test_WithSelectSubquery(t *testing.T) {
	query := With("foo", Select("id", "name").From("bar")).Select("name").From("foo").Build()
	assert.Equal(t, "WITH foo AS (SELECT id, name FROM bar) SELECT name FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_WithSelectMultiple(t *testing.T) {
	query := With("foo", Select("id", "name").From("bar")).With("xyz", "SELECT id, name FROM asdf").Select("name").From("foo").Build()
	assert.Equal(t, "WITH foo AS (SELECT id, name FROM bar), xyz AS (SELECT id, name FROM asdf) SELECT name FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}
