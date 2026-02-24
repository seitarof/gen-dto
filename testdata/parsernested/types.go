package parsernested

type Leaf struct {
	Value string
}

type Child struct {
	Leaf Leaf
}

type Root struct {
	Child     Child
	ChildPtr  *Child
	ChildList []Child
	Self      *Root
}
