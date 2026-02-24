package resolver

// NestedPairKey uniquely identifies one struct-pair conversion direction.
type NestedPairKey struct {
	SrcPkgPath string
	SrcName    string
	DstPkgPath string
	DstName    string
}

// NestedSet stores known nested struct pairings for fast lookup.
type NestedSet map[NestedPairKey]struct{}

// NestedAware can consume available nested struct pair information.
type NestedAware interface {
	SetNestedSet(NestedSet)
}
