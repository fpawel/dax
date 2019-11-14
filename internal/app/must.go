package app

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
