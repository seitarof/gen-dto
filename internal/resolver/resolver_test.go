package resolver

import (
	"go/types"
	"strings"
	"testing"

	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
)

func TestResolver_BasicCast(t *testing.T) {
	r := New(DefaultRules()...)
	pairs := []matcher.FieldPair{{
		SrcField: newBasicField("ID", "ID", "int", types.Typ[types.Int]),
		DstField: newBasicField("ID", "ID", "int64", types.Typ[types.Int64]),
	}}

	plans := r.Resolve(pairs, nil)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategyBasicCast {
		t.Fatalf("expected StrategyBasicCast, got %v", plans[0].Strategy)
	}
	if !strings.Contains(plans[0].Expression, "(int64)(src.ID)") {
		t.Fatalf("unexpected expression: %s", plans[0].Expression)
	}
}

func TestResolver_NullableToValue(t *testing.T) {
	r := New(DefaultRules()...)
	pairs := []matcher.FieldPair{{
		SrcField: newNamedStructField("Name", "Name", "sql.NullString", "database/sql", "NullString"),
		DstField: newBasicField("Name", "Name", "string", types.Typ[types.String]),
	}}

	plans := r.Resolve(pairs, nil)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategyNullableToValue {
		t.Fatalf("expected StrategyNullableToValue, got %v", plans[0].Strategy)
	}
}

func TestResolver_ValueToNullable(t *testing.T) {
	r := New(DefaultRules()...)
	pairs := []matcher.FieldPair{{
		SrcField: newBasicField("Name", "Name", "string", types.Typ[types.String]),
		DstField: newNamedStructField("Name", "Name", "sql.NullString", "database/sql", "NullString"),
	}}

	plans := r.Resolve(pairs, nil)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategyValueToNullable {
		t.Fatalf("expected StrategyValueToNullable, got %v", plans[0].Strategy)
	}
}

func TestResolver_PointerConversions(t *testing.T) {
	r := New(DefaultRules()...)

	unwrapPairs := []matcher.FieldPair{{
		SrcField: newPointerBasicField("Name", "Name", "string", types.Typ[types.String]),
		DstField: newBasicField("Name", "Name", "string", types.Typ[types.String]),
	}}
	unwrapPlans := r.Resolve(unwrapPairs, nil)
	if len(unwrapPlans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(unwrapPlans))
	}
	if unwrapPlans[0].Strategy != StrategyPointerUnwrap {
		t.Fatalf("expected StrategyPointerUnwrap, got %v", unwrapPlans[0].Strategy)
	}

	wrapPairs := []matcher.FieldPair{{
		SrcField: newBasicField("Name", "Name", "string", types.Typ[types.String]),
		DstField: newPointerBasicField("Name", "Name", "string", types.Typ[types.String]),
	}}
	wrapPlans := r.Resolve(wrapPairs, nil)
	if len(wrapPlans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(wrapPlans))
	}
	if wrapPlans[0].Strategy != StrategyPointerWrap {
		t.Fatalf("expected StrategyPointerWrap, got %v", wrapPlans[0].Strategy)
	}
}

func TestResolver_TimeStringConversions(t *testing.T) {
	r := New(DefaultRules()...)
	timeType := newNamedStructField("CreatedAt", "CreatedAt", "time.Time", "time", "Time")

	timeToString := []matcher.FieldPair{{
		SrcField: timeType,
		DstField: newBasicField("CreatedAt", "CreatedAt", "string", types.Typ[types.String]),
	}}
	plans := r.Resolve(timeToString, nil)
	if len(plans) != 1 || plans[0].Strategy != StrategyTimeToString {
		t.Fatalf("expected StrategyTimeToString, got %#v", plans)
	}

	stringToTime := []matcher.FieldPair{{
		SrcField: newBasicField("CreatedAt", "CreatedAt", "string", types.Typ[types.String]),
		DstField: timeType,
	}}
	plans = r.Resolve(stringToTime, nil)
	if len(plans) != 1 || plans[0].Strategy != StrategyStringToTime {
		t.Fatalf("expected StrategyStringToTime, got %#v", plans)
	}
}

func TestResolver_SliceConvert(t *testing.T) {
	r := New(DefaultRules()...)
	pairs := []matcher.FieldPair{{
		SrcField: newSliceBasicField("IDs", "IDs", "[]int", types.Typ[types.Int]),
		DstField: newSliceBasicField("IDs", "IDs", "[]int64", types.Typ[types.Int64]),
	}}

	plans := r.Resolve(pairs, nil)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategySliceConvert {
		t.Fatalf("expected StrategySliceConvert, got %v", plans[0].Strategy)
	}
}

func TestResolver_UnsupportedBecomesSkip(t *testing.T) {
	r := New(DefaultRules()...)

	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	pairs := []matcher.FieldPair{{
		SrcField: parser.FieldInfo{
			Name:       "Metadata",
			AccessPath: "Metadata",
			TypeStr:    "map[string]int",
			Type:       mapType,
			TypeInfo: parser.TypeDetail{
				Kind: parser.TypeKindMap,
			},
		},
		DstField: newBasicField("Metadata", "Metadata", "string", types.Typ[types.String]),
	}}

	plans := r.Resolve(pairs, nil)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategySkip {
		t.Fatalf("expected StrategySkip, got %v", plans[0].Strategy)
	}
}

func TestResolver_NestedStruct(t *testing.T) {
	r := New(DefaultRules()...)

	srcField := newNamedStructField("Address", "Address", "model.Address", "example.com/model", "Address")
	dstField := newNamedStructField("Address", "Address", "dto.AddressDTO", "example.com/dto", "AddressDTO")

	pairs := []matcher.FieldPair{{SrcField: srcField, DstField: dstField}}
	structPairs := []matcher.StructPair{
		{
			Src: &parser.StructInfo{Name: "Address", PkgPath: "example.com/model"},
			Dst: &parser.StructInfo{Name: "AddressDTO", PkgPath: "example.com/dto"},
		},
	}

	plans := r.Resolve(pairs, structPairs)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategyNestedStruct {
		t.Fatalf("expected StrategyNestedStruct, got %v", plans[0].Strategy)
	}
	if !strings.Contains(plans[0].Expression, "ConvertAddressToAddressDTO") {
		t.Fatalf("unexpected expression: %s", plans[0].Expression)
	}
}

func TestResolver_TypeAlias_DirectAssign(t *testing.T) {
	r := New(DefaultRules()...)

	apiPkg := types.NewPackage("example.com/api", "api")
	base := types.NewNamed(
		types.NewTypeName(0, apiPkg, "GetPatientProviderTypeResponse", nil),
		types.NewStruct(nil, nil),
		nil,
	)

	dtoPkg := types.NewPackage("example.com/dto", "dto")
	aliasName := types.NewTypeName(0, dtoPkg, "PatientProviderType", nil)
	alias := types.NewAlias(aliasName, base)

	pairs := []matcher.FieldPair{{
		SrcField: parser.FieldInfo{
			Name:       "ProviderType",
			AccessPath: "ProviderType",
			TypeStr:    "api.GetPatientProviderTypeResponse",
			Type:       base,
			TypeInfo: parser.TypeDetail{
				Kind:       parser.TypeKindStruct,
				PkgPath:    "example.com/api",
				StructName: "GetPatientProviderTypeResponse",
			},
		},
		DstField: parser.FieldInfo{
			Name:       "ProviderType",
			AccessPath: "ProviderType",
			TypeStr:    "PatientProviderType",
			Type:       alias,
			TypeInfo: parser.TypeDetail{
				Kind:       parser.TypeKindStruct,
				PkgPath:    "example.com/api",
				StructName: "GetPatientProviderTypeResponse",
			},
		},
	}}

	plans := r.Resolve(pairs, nil)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategyDirectAssign {
		t.Fatalf("expected StrategyDirectAssign, got %v", plans[0].Strategy)
	}
	if plans[0].Expression != "dst.ProviderType = src.ProviderType" {
		t.Fatalf("unexpected expression: %s", plans[0].Expression)
	}
}

func TestResolver_DoesNotUseConvertibleForPointerStructs(t *testing.T) {
	r := New(DefaultRules()...)

	srcPkg := types.NewPackage("example.com/src", "src")
	dstPkg := types.NewPackage("example.com/dst", "dst")
	srcNamed := types.NewNamed(
		types.NewTypeName(0, srcPkg, "Events", nil),
		types.NewStruct(nil, nil),
		nil,
	)
	dstNamed := types.NewNamed(
		types.NewTypeName(0, dstPkg, "Events", nil),
		types.NewStruct(nil, nil),
		nil,
	)

	srcPtr := types.NewPointer(srcNamed)
	dstPtr := types.NewPointer(dstNamed)

	srcElemDetail := parser.TypeDetail{
		Kind:       parser.TypeKindStruct,
		PkgPath:    "example.com/src",
		StructName: "Events",
	}
	dstElemDetail := parser.TypeDetail{
		Kind:       parser.TypeKindStruct,
		PkgPath:    "example.com/dst",
		StructName: "Events",
	}

	pairs := []matcher.FieldPair{{
		SrcField: parser.FieldInfo{
			Name:       "Events",
			AccessPath: "Events",
			TypeStr:    "*src.Events",
			Type:       srcPtr,
			TypeInfo: parser.TypeDetail{
				Kind:     parser.TypeKindPointer,
				ElemType: &srcElemDetail,
			},
		},
		DstField: parser.FieldInfo{
			Name:       "Events",
			AccessPath: "Events",
			TypeStr:    "*dst.Events",
			Type:       dstPtr,
			TypeInfo: parser.TypeDetail{
				Kind:     parser.TypeKindPointer,
				ElemType: &dstElemDetail,
			},
		},
	}}

	plans := r.Resolve(pairs, nil)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Strategy != StrategySkip {
		t.Fatalf("expected StrategySkip, got %v", plans[0].Strategy)
	}
}

func newBasicField(name, accessPath, typeStr string, t types.Type) parser.FieldInfo {
	return parser.FieldInfo{
		Name:       name,
		AccessPath: accessPath,
		TypeStr:    typeStr,
		Type:       t,
		TypeInfo: parser.TypeDetail{
			Kind:      parser.TypeKindBasic,
			IsBasic:   true,
			BasicKind: types.TypeString(t, nil),
			TypeName:  types.TypeString(t, nil),
		},
	}
}

func newNamedStructField(name, accessPath, typeStr, pkgPath, typeName string) parser.FieldInfo {
	pkg := types.NewPackage(pkgPath, "pkg")
	named := types.NewNamed(
		types.NewTypeName(0, pkg, typeName, nil),
		types.NewStruct(nil, nil),
		nil,
	)
	return parser.FieldInfo{
		Name:       name,
		AccessPath: accessPath,
		TypeStr:    typeStr,
		Type:       named,
		TypeInfo: parser.TypeDetail{
			Kind:       parser.TypeKindStruct,
			PkgPath:    pkgPath,
			StructName: typeName,
			TypeName:   pkgPath + "." + typeName,
		},
	}
}

func newPointerBasicField(name, accessPath, basicType string, elem types.Type) parser.FieldInfo {
	ptr := types.NewPointer(elem)
	elemInfo := parser.TypeDetail{
		Kind:      parser.TypeKindBasic,
		IsBasic:   true,
		BasicKind: basicType,
		TypeName:  basicType,
	}
	return parser.FieldInfo{
		Name:       name,
		AccessPath: accessPath,
		TypeStr:    "*" + basicType,
		Type:       ptr,
		TypeInfo: parser.TypeDetail{
			Kind:     parser.TypeKindPointer,
			ElemType: &elemInfo,
		},
	}
}

func newSliceBasicField(name, accessPath, typeStr string, elem types.Type) parser.FieldInfo {
	slice := types.NewSlice(elem)
	elemInfo := parser.TypeDetail{
		Kind:      parser.TypeKindBasic,
		IsBasic:   true,
		BasicKind: types.TypeString(elem, nil),
		TypeName:  types.TypeString(elem, nil),
	}
	return parser.FieldInfo{
		Name:       name,
		AccessPath: accessPath,
		TypeStr:    typeStr,
		Type:       slice,
		TypeInfo: parser.TypeDetail{
			Kind:     parser.TypeKindSlice,
			ElemType: &elemInfo,
		},
	}
}
