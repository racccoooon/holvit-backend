package h

type T2[Ta, Tb any] struct {
	First  Ta
	Second Tb
}

type T3[Ta, Tb, Tc any] struct {
	First  Ta
	Second Tb
	Third  Tc
}

type T4[Ta, Tb, Tc, Td any] struct {
	First  Ta
	Second Tb
	Third  Tc
	Fourth Td
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
