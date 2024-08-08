package utils

func PanicOnErr(f func() error) {
	err := f()
	if err != nil {
		panic(err)
	}
}
