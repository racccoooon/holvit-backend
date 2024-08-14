package sqlb

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_SelectConst(t *testing.T) {
	query := Select("true").Build()
	assert.Equal(t, query.Query, "SELECT true;")
	assert.Nil(t, query.Parameters)
}

func Test_SelectConstParam(t *testing.T) {
	query := Select("?, 1, 2, ?", true, false).Build()
}

func Test_SqlBuilder(t *testing.T) {
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

	cond1 := Predicate("foo.x == ?", 7)
	cond2 := Predicate("foo.y any ?", []int{1, 2, 3})
	cond3 := Predicate("foo.z > foo.asdf")

	cond := And(cond1, cond2)
	if someWhateverThing {
		cond = Or(cond, cond3)
	}

	Select("...").Where(cond)

	Select("...").Where("x in ?", Select("id").From("foos"))

	// select foo.id, foo.name, bar.something from foo join bar on foo.bar_id == bar.id and bar.asdf is null

	Select("foo.id", "foo.name", "bar.something").From("foo").Join("bar", "foo.bar_id == bar.id and bar.asdf is null and")

	// WITH names AS (SELECT name FROM people) SELECT x, y FROM asdf WHERE asdf.bar IN names

	With("names", Select("name").From("people")).Select("x", "y").From("asdf").Where("asdf.bar IN names")

}
