package source

type Address struct {
	City string
}

type User struct {
	ID      int
	Name    string
	Address Address
}
