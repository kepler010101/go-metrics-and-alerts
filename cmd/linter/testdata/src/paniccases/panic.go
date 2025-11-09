package paniccases

func bad() {
	panic("boom") // want "panic call is not allowed"
}

func good() {
	fn := func() {}
	fn()
}
