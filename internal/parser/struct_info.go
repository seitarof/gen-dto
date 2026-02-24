package parser

import "go/types"

// StructInfo holds flattened field information for one struct.
type StructInfo struct {
	Name    string
	PkgPath string
	PkgName string
	Fields  []FieldInfo
}

// FieldInfo stores one field mapping candidate.
type FieldInfo struct {
	Name       string
	AccessPath string
	TypeStr    string
	TypeInfo   TypeDetail
	Type       types.Type
	IsExported bool
	EmbedFrom  string
}

// TypeDetail keeps simplified type metadata for matching/resolution.
type TypeDetail struct {
	Kind       TypeKind
	PkgPath    string
	ElemType   *TypeDetail
	KeyType    *TypeDetail
	IsBasic    bool
	BasicKind  string
	StructName string
	TypeName   string
}

// TypeKind is coarse-grained type category.
type TypeKind int

const (
	TypeKindBasic TypeKind = iota
	TypeKindPointer
	TypeKindStruct
	TypeKindSlice
	TypeKindMap
	TypeKindInterface
	TypeKindOther
)
