package parserbasic

import "time"

type Profile struct {
	BirthAt time.Time
}

type User struct {
	ID      int
	Name    string
	Profile Profile
	Ptr     *Profile
	Tags    []string
	Scores  map[string]int
	hidden  string
}
