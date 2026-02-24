package parserembed

type Base struct {
	ID   int
	Name string
}

type InnerA struct {
	Code string
}

type InnerB struct {
	Code string
}

type User struct {
	Base
	InnerA
	InnerB
	Name   string
	Email  string
	hidden string
}
