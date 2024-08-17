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

func Test_SelectRawParam(t *testing.T) {
	query := Select(Raw("?", true)).Build()
	assert.Equal(t, "SELECT $1", query.Query)
	assert.Equal(t, []any{true}, query.Parameters)
}

func Test_SelectRawMultipleParams(t *testing.T) {
	query := Select(Raw("? + ?", 1, 2)).Build()
	assert.Equal(t, "SELECT $1 + $2", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
}

func Test_SelectRaw(t *testing.T) {
	query := Select(Raw("true")).Build()
	assert.Equal(t, "SELECT true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectMultipleRawsWithParams(t *testing.T) {
	query := Select(Raw("? + ?", 1, 2), Raw("?", true)).Build()
	assert.Equal(t, "SELECT $1 + $2, $3", query.Query)
	assert.Equal(t, []any{1, 2, true}, query.Parameters)
}

func Test_SelectRawWithQuotedStringAndParam(t *testing.T) {
	query := Select(Raw(`"foo?" + ?`, "bar")).Build()
	assert.Equal(t, `SELECT "foo?" + $1`, query.Query)
	assert.Equal(t, []any{"bar"}, query.Parameters)
}

func Test_SelectMultiple(t *testing.T) {
	query := Select("foo").Select("bar").Build()
	assert.Equal(t, "SELECT foo, bar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectAs(t *testing.T) {
	query := Select(As(Raw("? + ?", 1, 2), "bar"), As("foo", "foo2"), "a as b").Build()
	assert.Equal(t, "SELECT $1 + $2 AS bar, foo AS foo2, a as b", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
}

func Test_Subselect(t *testing.T) {
	query := Select(Raw("?", 1), As(Select(Raw("?", 2)), "b"), Raw("?", 3)).Build()
	assert.Equal(t, "SELECT $1, (SELECT $2) AS b, $3", query.Query)
	assert.Equal(t, []any{1, 2, 3}, query.Parameters)
}

func Test_SelectFrom(t *testing.T) {
	query := Select("foo", "bar").From("foobar").Build()
	assert.Equal(t, "SELECT foo, bar FROM foobar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFromAs(t *testing.T) {
	query := Select("foo", "bar").FromAs("foobar", "asdf").Build()
	assert.Equal(t, "SELECT foo, bar FROM foobar AS asdf", query.Query)
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
	query := Select("foo", "bar", Raw("?", 3)).From(Select("1 as foo", As(Raw("?", 2), "bar"))).Build()
	assert.Equal(t, "SELECT foo, bar, $1 FROM (SELECT 1 as foo, $2 AS bar)", query.Query)
	assert.Equal(t, []any{3, 2}, query.Parameters)
}

func Test_SelectJoin(t *testing.T) {
	query := Select("*").From("foo").Join("bar", "foo.id = bar.foo_id").Build()
	assert.Equal(t, "SELECT * FROM foo JOIN bar ON foo.id = bar.foo_id", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectMultipleJoinsParams(t *testing.T) {
	query := Select("*", Raw("?", 2)).From("foo").Join("bar", "foo.id = bar.foo_id and foo.x = ?", 3).Join("baz", "bar.id = baz.bar_id").Build()
	assert.Equal(t, "SELECT *, $1 FROM foo JOIN bar ON foo.id = bar.foo_id and foo.x = $2 JOIN baz ON bar.id = baz.bar_id", query.Query)
	assert.Equal(t, []any{2, 3}, query.Parameters)
}

func Test_SelectJoinAs(t *testing.T) {
	query := Select("*").From("foo").JoinAs("bar", "asdf", "foo.x = asdf.x").Build()
	assert.Equal(t, "SELECT * FROM foo JOIN bar AS asdf ON foo.x = asdf.x", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectInnerJoin(t *testing.T) {
	query := Select("*").From("foo").InnerJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo INNER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectInnerJoinAs(t *testing.T) {
	query := Select("*").From("foo").InnerJoinAs("bar", "asdf", "foo.x = asdf.x").Build()
	assert.Equal(t, "SELECT * FROM foo INNER JOIN bar AS asdf ON foo.x = asdf.x", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectLeftJoin(t *testing.T) {
	query := Select("*").From("foo").LeftJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo LEFT OUTER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectLeftJoinAs(t *testing.T) {
	query := Select("*").From("foo").LeftJoinAs("bar", "asdf", "foo.x = asdf.x").Build()
	assert.Equal(t, "SELECT * FROM foo LEFT OUTER JOIN bar AS asdf ON foo.x = asdf.x", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectRightJoin(t *testing.T) {
	query := Select("*").From("foo").RightJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo RIGHT OUTER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectRightJoinAs(t *testing.T) {
	query := Select("*").From("foo").RightJoinAs("bar", "asdf", "foo.x = asdf.x").Build()
	assert.Equal(t, "SELECT * FROM foo RIGHT OUTER JOIN bar AS asdf ON foo.x = asdf.x", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFullJoin(t *testing.T) {
	query := Select("*").From("foo").FullJoin("bar", "true").Build()
	assert.Equal(t, "SELECT * FROM foo FULL OUTER JOIN bar ON true", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectFullJoinAs(t *testing.T) {
	query := Select("*").From("foo").FullJoinAs("bar", "asdf", "foo.x = asdf.x").Build()
	assert.Equal(t, "SELECT * FROM foo FULL OUTER JOIN bar AS asdf ON foo.x = asdf.x", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectCrossJoin(t *testing.T) {
	query := Select("*").From("foo").CrossJoin("bar").Build()
	assert.Equal(t, "SELECT * FROM foo CROSS JOIN bar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectCrossJoinAs(t *testing.T) {
	query := Select("*").From("foo").CrossJoinAs("bar", "baz").Build()
	assert.Equal(t, "SELECT * FROM foo CROSS JOIN bar AS baz", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectJoinSubquery(t *testing.T) {
	query := Select("*").From("foo").JoinAs(Select("*").From("bar"), "asdf", "foo.x = asdf.x").Build()
	assert.Equal(t, "SELECT * FROM foo JOIN (SELECT * FROM bar) AS asdf ON foo.x = asdf.x", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectRawJoin(t *testing.T) {
	query := Select("*").From("foo").RawJoin("JOIN bar ON bar.id = ?", 1).Where("foo.x = ?", 2).Build()
	assert.Equal(t, "SELECT * FROM foo JOIN bar ON bar.id = $1 WHERE foo.x = $2", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
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
	query := Select("foo", Raw("?", 1)).From("bar").Where("x > ?", 2).Build()
	assert.Equal(t, "SELECT foo, $1 FROM bar WHERE x > $2", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
}

func Test_SelectWhereRaw(t *testing.T) {
	query := Select("*").From("foo").Where(And("true and false", "true")).Build()
	assert.Equal(t, "SELECT * FROM foo WHERE (true and false) AND (true)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectWhereRawComplex(t *testing.T) {
	query := Select("*").From("foo").Where(Or(
		And(
			Raw("x = ?", 1),
			"asdf < 10",
		),
		And(
			Raw("bar any ?", []int{2, 3, 4}),
			Not("z IS NULL"),
		),
	)).Build()
	assert.Equal(t, "SELECT * FROM foo WHERE ((x = $1) AND (asdf < 10)) OR ((bar any $2) AND (NOT(z IS NULL)))", query.Query)
	assert.Equal(t, []any{1, []int{2, 3, 4}}, query.Parameters)
}

func Test_NotImplicitRaw(t *testing.T) {
	q := Not("foo = bar").Build()
	assert.Equal(t, "NOT(foo = bar)", q.Query)
	assert.Empty(t, q.Parameters)
}

func Test_NotExplicitRaw(t *testing.T) {
	q := Not(Raw("foo = bar")).Build()
	assert.Equal(t, "NOT(foo = bar)", q.Query)
	assert.Empty(t, q.Parameters)
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
	query := Select("*").From("foo").Distinct(Raw("(a / ?)::int", 37)).Build()
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
	query := Select("*").From("foobar").OrderBy(Raw("CASE WHEN name = ? THEN foo ELSE ? END", "foobar", 7)).Build()
	assert.Equal(t, "SELECT * FROM foobar ORDER BY CASE WHEN name = $1 THEN foo ELSE $2 END", query.Query)
	assert.Equal(t, []any{"foobar", 7}, query.Parameters)
}

func Test_SelectLimitInt(t *testing.T) {
	query := Select("*").From("foobar").Limit(20).Build()
	assert.Equal(t, "SELECT * FROM foobar LIMIT $1", query.Query)
	assert.Equal(t, []any{20}, query.Parameters)
}

func Test_SelectLimitImplicitRaw(t *testing.T) {
	query := Select("*").From("foobar").Limit("foo").Build()
	assert.Equal(t, "SELECT * FROM foobar LIMIT foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectLimitRaw(t *testing.T) {
	query := Select("*").From("foobar").Limit(Raw("? + 2", 10)).Build()
	assert.Equal(t, "SELECT * FROM foobar LIMIT $1 + 2", query.Query)
	assert.Equal(t, []any{10}, query.Parameters)
}

func Test_SelectLimitSubquery(t *testing.T) {
	query := Select("*").From("foo").Limit(Select("count(*)").From("bar")).Build()
	assert.Equal(t, "SELECT * FROM foo LIMIT (SELECT count(*) FROM bar)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectOffsetInt(t *testing.T) {
	query := Select("*").From("foobar").Offset(20).Build()
	assert.Equal(t, "SELECT * FROM foobar OFFSET $1", query.Query)
	assert.Equal(t, []any{20}, query.Parameters)
}

func Test_SelectOffsetImplicitRaw(t *testing.T) {
	query := Select("*").From("foobar").Offset("foo").Build()
	assert.Equal(t, "SELECT * FROM foobar OFFSET foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectOffsetRaw(t *testing.T) {
	query := Select("*").From("foobar").Offset(Raw("? + 2", 10)).Build()
	assert.Equal(t, "SELECT * FROM foobar OFFSET $1 + 2", query.Query)
	assert.Equal(t, []any{10}, query.Parameters)
}

func Test_SelectOffsetSubquery(t *testing.T) {
	query := Select("*").From("foo").Offset(Select("count(*)").From("bar")).Build()
	assert.Equal(t, "SELECT * FROM foo OFFSET (SELECT count(*) FROM bar)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectLimitOffset(t *testing.T) {
	query := Select("*").From("foobar").Limit(10).Offset(20).Build()
	assert.Equal(t, "SELECT * FROM foobar LIMIT $1 OFFSET $2", query.Query)
	assert.Equal(t, []any{10, 20}, query.Parameters)
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

func Test_WithInsert(t *testing.T) {
	query := With("foo", "SELECT * FROM bar").InsertInto("foobar", "a", "b").Query(Raw("SELECT a, b FROM foo")).Build()
	assert.Equal(t, "WITH foo AS (SELECT * FROM bar) INSERT INTO foobar (a, b) SELECT a, b FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_WithUpdate(t *testing.T) {
	query := With("foo", "SELECT * FROM bar").Update("foobar").Set("x", 1).Build()
	assert.Equal(t, "WITH foo AS (SELECT * FROM bar) UPDATE foobar SET x = $1", query.Query)
	assert.Equal(t, []any{1}, query.Parameters)
}

func Test_WithDelete(t *testing.T) {
	query := With("ids", "SELECT id FROM bar").DeleteFrom("foobar").Where("id in (?)", Raw("SELECT id FROM ids")).Build()
	assert.Equal(t, "WITH ids AS (SELECT id FROM bar) DELETE FROM foobar WHERE id in (SELECT id FROM ids)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_WithRaw(t *testing.T) {
	query := With("foo", "SELECT * FROM bar WHERE x = ?", 1).Raw("SELECT * FROM foo WHERE y = ?", 2).Build()
	assert.Equal(t, "WITH foo AS (SELECT * FROM bar WHERE x = $1) SELECT * FROM foo WHERE y = $2", query.Query)
	assert.Equal(t, []any{1, 2}, query.Parameters)
}

func Test_WeirdQuestionMarkPosition(t *testing.T) {
	query := Select("*").From("s").Where("s.name = any(?::text[])", 1).Build()
	assert.Equal(t, "SELECT * FROM s WHERE s.name = any($1::text[])", query.Query)
	assert.Equal(t, []any{1}, query.Parameters)
}

func Test_StringGetsTokenizedCorrectly(t *testing.T) {
	query := Select("*").From("foobar").Where("foobar.name = 'as?df'").Build()
	assert.Equal(t, "SELECT * FROM foobar WHERE foobar.name = 'as?df'", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectGroupBy(t *testing.T) {
	query := Select("*").From("foobar").GroupBy("asdf").Build()
	assert.Equal(t, "SELECT * FROM foobar GROUP BY asdf", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectGroupByMultiple(t *testing.T) {
	query := Select("*").From("foobar").GroupBy("asdf", "baz").Build()
	assert.Equal(t, "SELECT * FROM foobar GROUP BY asdf, baz", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_SelectHaving(t *testing.T) {
	query := Select("*").From("foobar").GroupBy("asdf").Having("sum(x) > ?", 10).Build()
	assert.Equal(t, "SELECT * FROM foobar GROUP BY asdf HAVING sum(x) > $1", query.Query)
	assert.Equal(t, []any{10}, query.Parameters)
}

func Test_SelectHavingMultiple(t *testing.T) {
	query := Select("*").From("foobar").GroupBy("asdf").Having("sum(x) > ?", 10).Having("avg(x) < ?", 20).Build()
	assert.Equal(t, "SELECT * FROM foobar GROUP BY asdf HAVING (sum(x) > $1) AND (avg(x) < $2)", query.Query)
	assert.Equal(t, []any{10, 20}, query.Parameters)
}

func Test_InsertMultiple(t *testing.T) {
	query := InsertInto("foo", "a", "b", "c").Values(1, 2, 3).Values(2, 3, 4).Values(3, 4, 5).Build()
	assert.Equal(t, "INSERT INTO foo (a, b, c) VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)", query.Query)
	assert.Equal(t, []any{1, 2, 3, 2, 3, 4, 3, 4, 5}, query.Parameters)
}

func Test_InsertQuery(t *testing.T) {
	query := InsertInto("foo", "a", "b", "c").Query(Select("a", "b", "c").From("bar")).Build()
	assert.Equal(t, "INSERT INTO foo (a, b, c) SELECT a, b, c FROM bar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_InsertRaws(t *testing.T) {
	query := InsertInto("foo", "a", "b", "c").Values(Raw("gen_random_uuid()"), 123, Select("COUNT(*)").From("foobar")).Build()
	assert.Equal(t, "INSERT INTO foo (a, b, c) VALUES (gen_random_uuid(), $1, (SELECT COUNT(*) FROM foobar))", query.Query)
	assert.Equal(t, []any{123}, query.Parameters)
}

func Test_InsertReturning(t *testing.T) {
	query := InsertInto("foo", "a", "b", "c").Values(1, 2, 3).Returning("id", "bar", Raw("x + ? AS y", 4)).Build()
	assert.Equal(t, "INSERT INTO foo (a, b, c) VALUES ($1, $2, $3) RETURNING id, bar, x + $4 AS y", query.Query)
	assert.Equal(t, []any{1, 2, 3, 4}, query.Parameters)
}

func Test_Update(t *testing.T) {
	query := Update("foo").Set("a", 10).Set("b", "foo").Set("c", Raw("gen_random_uuid()")).Build()
	assert.Equal(t, "UPDATE foo SET a = $1, b = $2, c = gen_random_uuid()", query.Query)
	assert.Equal(t, []any{10, "foo"}, query.Parameters)
}

func Test_UpdateMultipleColumnsFromSubquery(t *testing.T) {
	query := Update("foo").Set("(a, b)", Select("x", "y").From("bar").Where("foo.id = bar.id")).Build()
	assert.Equal(t, "UPDATE foo SET (a, b) = (SELECT x, y FROM bar WHERE foo.id = bar.id)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_UpdateWhere(t *testing.T) {
	query := Update("foo").Set("deleted", true).Where("id = ?", 123).Build()
	assert.Equal(t, "UPDATE foo SET deleted = $1 WHERE id = $2", query.Query)
	assert.Equal(t, []any{true, 123}, query.Parameters)
}

func Test_UpdateFrom(t *testing.T) {
	query := Update("foo").Set("foo.a", Raw("bar.b")).From("bar").Where("foo.id = bar.id").Build()
	assert.Equal(t, "UPDATE foo SET foo.a = bar.b FROM bar WHERE foo.id = bar.id", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_UpdateFromMultiple(t *testing.T) {
	query := Update("foo").Set("foo.a", Raw("bar.b + baz.c")).From("bar").From("baz").Where("foo.id = bar.id").Where("foo.id = baz.id").Build()
	assert.Equal(t, "UPDATE foo SET foo.a = bar.b + baz.c FROM bar, baz WHERE (foo.id = bar.id) AND (foo.id = baz.id)", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_UpdateFromSubquery(t *testing.T) {
	query := Update("foo").Set("foo.a", Raw("baz.c")).From(As(Select("id", "x as c").From("bar"), "baz")).Where("foo.id = baz.id").Build()
	assert.Equal(t, "UPDATE foo SET foo.a = baz.c FROM (SELECT id, x as c FROM bar) AS baz WHERE foo.id = baz.id", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_UpdateReturning(t *testing.T) {
	query := Update("foo").Set("a", 123).Returning("id").Build()
	assert.Equal(t, "UPDATE foo SET a = $1 RETURNING id", query.Query)
	assert.Equal(t, []any{123}, query.Parameters)
}

func Test_Delete(t *testing.T) {
	query := DeleteFrom("foo").Build()
	assert.Equal(t, "DELETE FROM foo", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_DeleteWhere(t *testing.T) {
	query := DeleteFrom("foo").Where("x = ?", 1).Build()
	assert.Equal(t, "DELETE FROM foo WHERE x = $1", query.Query)
	assert.Equal(t, []any{1}, query.Parameters)
}

func Test_DeleteUsing(t *testing.T) {
	query := DeleteFrom("foo").Using("bar").Where("foo.id = bar.id").Where("bar.x = ?", 1).Build()
	assert.Equal(t, "DELETE FROM foo USING bar WHERE (foo.id = bar.id) AND (bar.x = $1)", query.Query)
	assert.Equal(t, []any{1}, query.Parameters)
}

func Test_DeleteUsingMultiple(t *testing.T) {
	query := DeleteFrom("foo").Using("bar").Using("baz").Where("foo.id = bar.id").Where("bar.x = ?", 1).Build()
	assert.Equal(t, "DELETE FROM foo USING bar, baz WHERE (foo.id = bar.id) AND (bar.x = $1)", query.Query)
	assert.Equal(t, []any{1}, query.Parameters)
}

func Test_DeleteReturning(t *testing.T) {
	query := DeleteFrom("foo").Where("x = true").Returning("*").Build()
	assert.Equal(t, "DELETE FROM foo WHERE x = true RETURNING *", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_RawSubquery(t *testing.T) {
	query := Raw("SELECT * FROM foo WHERE id IN ? AND x = ?", Select("foo_id").From("foobar"), 123).Build()
	assert.Equal(t, "SELECT * FROM foo WHERE id IN (SELECT foo_id FROM foobar) AND x = $1", query.Query)
	assert.Equal(t, []any{123}, query.Parameters)
}

func Test_NestedRawWeirdness(t *testing.T) {
	query := Raw("? ? ? ? ? ? ? ?", Raw("SELECT"), Raw("foo"), Raw("FROM"), Raw("bar"), Raw("WHERE"), Raw("x"), Raw("="), 1).Build()
	assert.Equal(t, "SELECT foo FROM bar WHERE x = $1", query.Query)
	assert.Equal(t, []any{1}, query.Parameters)
}

func Test_DeeplyNestedRaw(t *testing.T) {
	query := Raw("? ?",
		Raw("SELECT"),
		Raw("? ?",
			Raw("foo"),
			Raw("? ?",
				Raw("FROM"),
				Raw("? ?",
					Raw("bar"),
					Raw("? ?",
						Raw("WHERE"),
						Raw("x = ?", 1)))))).Build()
	assert.Equal(t, "SELECT foo FROM bar WHERE x = $1", query.Query)
	assert.Equal(t, []any{1}, query.Parameters)
}

func Test_AndSingleTerm(t *testing.T) {
	query := And("foo = bar").Build()
	assert.Equal(t, "foo = bar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_OrSingleTerm(t *testing.T) {
	query := Or("foo = bar").Build()
	assert.Equal(t, "foo = bar", query.Query)
	assert.Empty(t, query.Parameters)
}

func Test_EmptyOrPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = Or()
	})
}

func Test_EmptyAndPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = And()
	})
}

func Test_InsertQueryAndValuesPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = InsertInto("foo").Values(1, 2, 3).Query(Raw("bar"))
	})
	assert.Panics(t, func() {
		_ = InsertInto("foo").Query(Raw("bar")).Values(1, 2, 3)
	})
}

func Test_SubqueryInvalidTypePanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = Select("*").From(1)
	})
}
