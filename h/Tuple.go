package h

type T2[Ta, Tb any] struct {
	First  Ta
	Second Tb
}

func (t T2[Ta, Tb]) Values() (Ta, Tb) {
	return t.First, t.Second
}

type T3[Ta, Tb, Tc any] struct {
	First  Ta
	Second Tb
	Third  Tc
}

func (t T3[Ta, Tb, Tc]) Values() (Ta, Tb, Tc) {
	return t.First, t.Second, t.Third
}

type T4[Ta, Tb, Tc, Td any] struct {
	First  Ta
	Second Tb
	Third  Tc
	Fourth Td
}

func (t T4[Ta, Tb, Tc, Td]) Values() (Ta, Tb, Tc, Td) {
	return t.First, t.Second, t.Third, t.Fourth
}

func NewT2[Ta, Tb any](first Ta, second Tb) T2[Ta, Tb] {
	return T2[Ta, Tb]{First: first, Second: second}
}

func NewT3[Ta, Tb, Tc any](first Ta, second Tb, third Tc) T3[Ta, Tb, Tc] {
	return T3[Ta, Tb, Tc]{First: first, Second: second, Third: third}
}

func NewT4[Ta, Tb, Tc, Td any](first Ta, second Tb, third Tc, fourth Td) T4[Ta, Tb, Tc, Td] {
	return T4[Ta, Tb, Tc, Td]{First: first, Second: second, Third: third, Fourth: fourth}
}
