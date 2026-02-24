package dest

type Address struct {
	City string
}

type UserResponse struct {
	ID      int64
	Name    string
	Address Address
}
