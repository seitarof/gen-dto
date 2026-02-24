package resolver

import (
	"go/types"
	"strings"

	"github.com/seitarof/gen-dto/internal/parser"
)

// DefaultRules returns built-in rules in priority order.
func DefaultRules() []Rule {
	return []Rule{
		&SameTypeRule{},
		&BasicCastRule{},
		&PointerRule{},
		&NullableRule{},
		&TimeStringRule{},
		&NestedStructRule{},
		&SliceConvertRule{},
		&StringerRule{},
		&AssignableRule{},
		&ConvertibleRule{},
	}
}

// SameTypeRule: identical type -> direct assignment.
type SameTypeRule struct{}

func (r *SameTypeRule) Name() string { return "same-type" }

func (r *SameTypeRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	if isIdenticalType(src.Type, dst.Type) {
		return newPlan(src, dst, StrategyDirectAssign, assign(dstSelector(dst), srcSelector(src))), true
	}
	return ConversionPlan{}, false
}

// BasicCastRule: basic/alias basic conversion with cast.
type BasicCastRule struct{}

func (r *BasicCastRule) Name() string { return "basic-cast" }

func (r *BasicCastRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	if !src.TypeInfo.IsBasic || !dst.TypeInfo.IsBasic {
		return ConversionPlan{}, false
	}
	if isIdenticalType(src.Type, dst.Type) {
		return ConversionPlan{}, false
	}
	if !types.ConvertibleTo(src.Type, dst.Type) {
		return ConversionPlan{}, false
	}
	expr := castAssign(dstSelector(dst), dst.TypeStr, srcSelector(src))
	return newPlan(src, dst, StrategyBasicCast, expr), true
}

// PointerRule: pointer <-> value conversion for non-nested types.
type PointerRule struct{}

func (r *PointerRule) Name() string { return "pointer" }

func (r *PointerRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	srcElem, srcPtr := pointerElem(src.Type)
	dstElem, dstPtr := pointerElem(dst.Type)

	if srcPtr && !dstPtr {
		if isIdenticalType(srcElem, dst.Type) {
			srcSel := srcSelector(src)
			dstSel := dstSelector(dst)
			expr := "if " + srcSel + " != nil {\n" + dstSel + " = *" + srcSel + "\n}"
			return newPlan(src, dst, StrategyPointerUnwrap, expr), true
		}
		if src.TypeInfo.ElemType != nil && src.TypeInfo.ElemType.Kind == parser.TypeKindStruct {
			return ConversionPlan{}, false
		}
		if types.ConvertibleTo(srcElem, dst.Type) && src.TypeInfo.ElemType != nil && src.TypeInfo.ElemType.IsBasic && dst.TypeInfo.IsBasic {
			srcSel := srcSelector(src)
			dstSel := dstSelector(dst)
			expr := "if " + srcSel + " != nil {\n" + dstSel + " = " + dst.TypeStr + "(*" + srcSel + ")\n}"
			return newPlan(src, dst, StrategyPointerUnwrap, expr), true
		}
	}

	if !srcPtr && dstPtr {
		if isIdenticalType(src.Type, dstElem) {
			expr := assign(dstSelector(dst), "&"+srcSelector(src))
			return newPlan(src, dst, StrategyPointerWrap, expr), true
		}
		if dst.TypeInfo.ElemType != nil && dst.TypeInfo.ElemType.Kind == parser.TypeKindStruct {
			return ConversionPlan{}, false
		}
		if types.ConvertibleTo(src.Type, dstElem) && src.TypeInfo.IsBasic && dst.TypeInfo.ElemType != nil && dst.TypeInfo.ElemType.IsBasic {
			dstElemType := strings.TrimPrefix(dst.TypeStr, "*")
			srcSel := srcSelector(src)
			dstSel := dstSelector(dst)
			expr := "{\nv := " + dstElemType + "(" + srcSel + ")\n" + dstSel + " = &v\n}"
			return newPlan(src, dst, StrategyPointerWrap, expr), true
		}
	}

	return ConversionPlan{}, false
}

// NullableRule handles database/sql.Null* <-> value conversions.
type NullableRule struct{}

func (r *NullableRule) Name() string { return "nullable" }

func (r *NullableRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	srcMeta, srcIsNullable := nullableTypeMeta(src.TypeInfo)
	dstMeta, dstIsNullable := nullableTypeMeta(dst.TypeInfo)

	if srcIsNullable && typeMatchesCanonical(dst.TypeInfo, srcMeta.valueType) {
		srcSel := srcSelector(src)
		dstSel := dstSelector(dst)
		expr := "if " + srcSel + ".Valid {\n" + dstSel + " = " + srcSel + "." + srcMeta.valueField + "\n}"
		return newPlan(src, dst, StrategyNullableToValue, expr), true
	}

	if dstIsNullable && typeMatchesCanonical(src.TypeInfo, dstMeta.valueType) {
		srcSel := srcSelector(src)
		validExpr := dstMeta.validExpr(srcSel)
		expr := dstSelector(dst) + " = " + dst.TypeStr + "{" + dstMeta.valueField + ": " + srcSel + ", Valid: " + validExpr + "}"
		return newPlan(src, dst, StrategyValueToNullable, expr), true
	}

	return ConversionPlan{}, false
}

// TimeStringRule handles time.Time <-> string.
type TimeStringRule struct{}

func (r *TimeStringRule) Name() string { return "time-string" }

func (r *TimeStringRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	if isTimeType(src.TypeInfo) && isStringType(dst.TypeInfo) {
		expr := assign(dstSelector(dst), srcSelector(src)+".Format(time.RFC3339)")
		return newPlan(src, dst, StrategyTimeToString, expr), true
	}
	if isStringType(src.TypeInfo) && isTimeType(dst.TypeInfo) {
		srcSel := srcSelector(src)
		dstSel := dstSelector(dst)
		expr := "if parsed, err := time.Parse(time.RFC3339, " + srcSel + "); err == nil {\n" + dstSel + " = parsed\n}"
		return newPlan(src, dst, StrategyStringToTime, expr), true
	}
	return ConversionPlan{}, false
}

// NestedStructRule maps nested struct fields via generated converters.
type NestedStructRule struct {
	nestedSet NestedSet
}

func (r *NestedStructRule) Name() string { return "nested-struct" }

func (r *NestedStructRule) SetNestedSet(nestedSet NestedSet) {
	r.nestedSet = nestedSet
}

func (r *NestedStructRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	if len(r.nestedSet) == 0 {
		return ConversionPlan{}, false
	}

	if srcRef, ok := structRefFromDetail(src.TypeInfo); ok {
		if dstRef, ok := structRefFromDetail(dst.TypeInfo); ok {
			if _, ok := r.nestedSet[pairKey(srcRef, dstRef)]; !ok {
				return ConversionPlan{}, false
			}
			fn := DefaultConverterName(srcRef.pkgPath, srcRef.name, dstRef.pkgPath, dstRef.name)
			srcSel := srcSelector(src)
			dstSel := dstSelector(dst)
			expr := "if v := " + fn + "(&" + srcSel + "); v != nil {\n" + dstSel + " = *v\n}"
			return newPlan(src, dst, StrategyNestedStruct, expr), true
		}
	}

	if srcRef, ok := ptrStructRef(src.TypeInfo); ok {
		if dstRef, ok := ptrStructRef(dst.TypeInfo); ok {
			if _, ok := r.nestedSet[pairKey(srcRef, dstRef)]; !ok {
				return ConversionPlan{}, false
			}
			fn := DefaultConverterName(srcRef.pkgPath, srcRef.name, dstRef.pkgPath, dstRef.name)
			expr := assign(dstSelector(dst), fn+"("+srcSelector(src)+")")
			return newPlan(src, dst, StrategyNestedStructPtr, expr), true
		}
	}

	if srcRef, ok := ptrStructRef(src.TypeInfo); ok {
		if dstRef, ok := structRefFromDetail(dst.TypeInfo); ok {
			if _, ok := r.nestedSet[pairKey(srcRef, dstRef)]; !ok {
				return ConversionPlan{}, false
			}
			fn := DefaultConverterName(srcRef.pkgPath, srcRef.name, dstRef.pkgPath, dstRef.name)
			srcSel := srcSelector(src)
			dstSel := dstSelector(dst)
			expr := "if v := " + fn + "(" + srcSel + "); v != nil {\n" + dstSel + " = *v\n}"
			return newPlan(src, dst, StrategyNestedStructPtr, expr), true
		}
	}

	if srcRef, ok := structRefFromDetail(src.TypeInfo); ok {
		if dstRef, ok := ptrStructRef(dst.TypeInfo); ok {
			if _, ok := r.nestedSet[pairKey(srcRef, dstRef)]; !ok {
				return ConversionPlan{}, false
			}
			fn := DefaultConverterName(srcRef.pkgPath, srcRef.name, dstRef.pkgPath, dstRef.name)
			expr := assign(dstSelector(dst), fn+"(&"+srcSelector(src)+")")
			return newPlan(src, dst, StrategyNestedStructPtr, expr), true
		}
	}

	if srcRef, ok := sliceStructRef(src.TypeInfo); ok {
		if dstRef, ok := sliceStructRef(dst.TypeInfo); ok {
			if _, ok := r.nestedSet[pairKey(srcRef, dstRef)]; !ok {
				return ConversionPlan{}, false
			}
			fn := DefaultConverterName(srcRef.pkgPath, srcRef.name, dstRef.pkgPath, dstRef.name)
			srcSel := srcSelector(src)
			dstSel := dstSelector(dst)
			expr := "if " + srcSel + " != nil {\n" +
				dstSel + " = make(" + dst.TypeStr + ", len(" + srcSel + "))\n" +
				"for i := range " + srcSel + " {\n" +
				"if v := " + fn + "(&" + srcSel + "[i]); v != nil {\n" +
				dstSel + "[i] = *v\n}\n}\n}"
			return newPlan(src, dst, StrategyNestedSlice, expr), true
		}
	}

	return ConversionPlan{}, false
}

// SliceConvertRule handles []A -> []B element casts.
type SliceConvertRule struct{}

func (r *SliceConvertRule) Name() string { return "slice-convert" }

func (r *SliceConvertRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	srcElemType, ok := sliceElem(src.Type)
	if !ok {
		return ConversionPlan{}, false
	}
	dstElemType, ok := sliceElem(dst.Type)
	if !ok {
		return ConversionPlan{}, false
	}
	if src.TypeInfo.ElemType == nil || dst.TypeInfo.ElemType == nil {
		return ConversionPlan{}, false
	}
	if src.TypeInfo.ElemType.Kind == parser.TypeKindStruct || dst.TypeInfo.ElemType.Kind == parser.TypeKindStruct {
		return ConversionPlan{}, false
	}
	if !(types.Identical(srcElemType, dstElemType) || types.ConvertibleTo(srcElemType, dstElemType)) {
		return ConversionPlan{}, false
	}

	dstElemTypeStr := strings.TrimPrefix(dst.TypeStr, "[]")
	srcSel := srcSelector(src)
	dstSel := dstSelector(dst)
	assignExpr := dstSel + "[i] = " + srcSel + "[i]"
	strategy := StrategyDirectAssign
	if !types.Identical(srcElemType, dstElemType) {
		assignExpr = dstSel + "[i] = " + dstElemTypeStr + "(" + srcSel + "[i])"
		strategy = StrategySliceConvert
	}

	expr := "if " + srcSel + " != nil {\n" +
		dstSel + " = make(" + dst.TypeStr + ", len(" + srcSel + "))\n" +
		"for i := range " + srcSel + " {\n" +
		assignExpr + "\n}\n}"
	return newPlan(src, dst, strategy, expr), true
}

// StringerRule calls String() for string destinations.
type StringerRule struct{}

func (r *StringerRule) Name() string { return "stringer" }

func (r *StringerRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	if !isStringType(dst.TypeInfo) || !hasStringMethod(src.Type) {
		return ConversionPlan{}, false
	}
	if _, ok := pointerElem(src.Type); ok {
		srcSel := srcSelector(src)
		dstSel := dstSelector(dst)
		expr := "if " + srcSel + " != nil {\n" + dstSel + " = " + srcSel + ".String()\n}"
		return newPlan(src, dst, StrategyCustomFunc, expr), true
	}
	expr := assign(dstSelector(dst), srcSelector(src)+".String()")
	return newPlan(src, dst, StrategyCustomFunc, expr), true
}

// AssignableRule uses types.AssignableTo.
type AssignableRule struct{}

func (r *AssignableRule) Name() string { return "assignable" }

func (r *AssignableRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	if types.AssignableTo(src.Type, dst.Type) {
		return newPlan(src, dst, StrategyDirectAssign, assign(dstSelector(dst), srcSelector(src))), true
	}
	return ConversionPlan{}, false
}

// ConvertibleRule uses types.ConvertibleTo.
type ConvertibleRule struct{}

func (r *ConvertibleRule) Name() string { return "convertible" }

func (r *ConvertibleRule) Try(src, dst parser.FieldInfo) (ConversionPlan, bool) {
	if !isConvertibleFallbackKind(src.TypeInfo.Kind) || !isConvertibleFallbackKind(dst.TypeInfo.Kind) {
		return ConversionPlan{}, false
	}
	if types.ConvertibleTo(src.Type, dst.Type) {
		expr := castAssign(dstSelector(dst), dst.TypeStr, srcSelector(src))
		return newPlan(src, dst, StrategyBasicCast, expr), true
	}
	return ConversionPlan{}, false
}

type nullableMeta struct {
	valueType  string
	valueField string
	validExpr  func(srcExpr string) string
}

var nullableMapping = map[string]nullableMeta{
	"database/sql.NullString": {
		valueType:  "string",
		valueField: "String",
		validExpr: func(srcExpr string) string {
			return srcExpr + ` != ""`
		},
	},
	"database/sql.NullInt64": {
		valueType:  "int64",
		valueField: "Int64",
		validExpr:  func(string) string { return "true" },
	},
	"database/sql.NullInt32": {
		valueType:  "int32",
		valueField: "Int32",
		validExpr:  func(string) string { return "true" },
	},
	"database/sql.NullInt16": {
		valueType:  "int16",
		valueField: "Int16",
		validExpr:  func(string) string { return "true" },
	},
	"database/sql.NullFloat64": {
		valueType:  "float64",
		valueField: "Float64",
		validExpr:  func(string) string { return "true" },
	},
	"database/sql.NullBool": {
		valueType:  "bool",
		valueField: "Bool",
		validExpr:  func(string) string { return "true" },
	},
	"database/sql.NullByte": {
		valueType:  "byte",
		valueField: "Byte",
		validExpr:  func(string) string { return "true" },
	},
	"database/sql.NullTime": {
		valueType:  "time.Time",
		valueField: "Time",
		validExpr: func(srcExpr string) string {
			return "!" + srcExpr + ".IsZero()"
		},
	},
}

func newPlan(src, dst parser.FieldInfo, strategy ConversionStrategy, expression string) ConversionPlan {
	return ConversionPlan{
		SrcField:   src,
		DstField:   dst,
		Strategy:   strategy,
		Expression: expression,
	}
}

func assign(dst, src string) string {
	return dst + " = " + src
}

func castAssign(dst, dstType, src string) string {
	return dst + " = (" + dstType + ")(" + src + ")"
}

func isConvertibleFallbackKind(kind parser.TypeKind) bool {
	switch kind {
	case parser.TypeKindBasic, parser.TypeKindStruct:
		return true
	default:
		return false
	}
}

func srcSelector(f parser.FieldInfo) string {
	return "src." + f.AccessPath
}

func dstSelector(f parser.FieldInfo) string {
	return "dst." + f.AccessPath
}

func isIdenticalType(src, dst types.Type) bool {
	if src == nil || dst == nil {
		return false
	}
	if types.Identical(src, dst) {
		return true
	}
	srcUnaliased := types.Unalias(src)
	dstUnaliased := types.Unalias(dst)
	if types.Identical(srcUnaliased, dstUnaliased) {
		return true
	}
	if types.TypeString(srcUnaliased, nil) == types.TypeString(dstUnaliased, nil) {
		return true
	}
	return types.TypeString(src, nil) == types.TypeString(dst, nil)
}

func pointerElem(t types.Type) (types.Type, bool) {
	switch v := t.(type) {
	case *types.Alias:
		return pointerElem(v.Rhs())
	case *types.Pointer:
		return v.Elem(), true
	case *types.Named:
		if p, ok := v.Underlying().(*types.Pointer); ok {
			return p.Elem(), true
		}
	}
	return nil, false
}

func sliceElem(t types.Type) (types.Type, bool) {
	switch v := t.(type) {
	case *types.Alias:
		return sliceElem(v.Rhs())
	case *types.Slice:
		return v.Elem(), true
	case *types.Named:
		if s, ok := v.Underlying().(*types.Slice); ok {
			return s.Elem(), true
		}
	}
	return nil, false
}

func nullableTypeMeta(detail parser.TypeDetail) (nullableMeta, bool) {
	if detail.Kind != parser.TypeKindStruct || detail.PkgPath == "" || detail.StructName == "" {
		return nullableMeta{}, false
	}
	meta, ok := nullableMapping[detail.PkgPath+"."+detail.StructName]
	return meta, ok
}

func typeMatchesCanonical(detail parser.TypeDetail, canonical string) bool {
	if strings.Contains(canonical, ".") {
		parts := strings.SplitN(canonical, ".", 2)
		return detail.Kind == parser.TypeKindStruct && detail.PkgPath == parts[0] && detail.StructName == parts[1]
	}
	if canonical == "byte" {
		return detail.IsBasic && (detail.BasicKind == "byte" || detail.BasicKind == "uint8")
	}
	return detail.IsBasic && detail.BasicKind == canonical
}

func isStringType(detail parser.TypeDetail) bool {
	return detail.IsBasic && detail.BasicKind == "string"
}

func isTimeType(detail parser.TypeDetail) bool {
	return detail.Kind == parser.TypeKindStruct && detail.PkgPath == "time" && detail.StructName == "Time"
}

type structRef struct {
	pkgPath string
	name    string
}

func pairKey(src, dst structRef) NestedPairKey {
	return NestedPairKey{
		SrcPkgPath: src.pkgPath,
		SrcName:    src.name,
		DstPkgPath: dst.pkgPath,
		DstName:    dst.name,
	}
}

func structRefFromDetail(detail parser.TypeDetail) (structRef, bool) {
	if detail.Kind != parser.TypeKindStruct || detail.PkgPath == "" || detail.StructName == "" {
		return structRef{}, false
	}
	return structRef{pkgPath: detail.PkgPath, name: detail.StructName}, true
}

func ptrStructRef(detail parser.TypeDetail) (structRef, bool) {
	if detail.Kind != parser.TypeKindPointer || detail.ElemType == nil {
		return structRef{}, false
	}
	return structRefFromDetail(*detail.ElemType)
}

func sliceStructRef(detail parser.TypeDetail) (structRef, bool) {
	if detail.Kind != parser.TypeKindSlice || detail.ElemType == nil {
		return structRef{}, false
	}
	if ref, ok := structRefFromDetail(*detail.ElemType); ok {
		return ref, true
	}
	return ptrStructRef(*detail.ElemType)
}

func hasStringMethod(t types.Type) bool {
	if t == nil {
		return false
	}
	if hasStringMethodOnType(t) {
		return true
	}
	if _, ok := t.(*types.Pointer); ok {
		return false
	}
	return hasStringMethodOnType(types.NewPointer(t))
}

func hasStringMethodOnType(t types.Type) bool {
	ms := types.NewMethodSet(t)
	for i := 0; i < ms.Len(); i++ {
		sel := ms.At(i)
		if sel.Obj().Name() != "String" {
			continue
		}
		sig, ok := sel.Obj().Type().(*types.Signature)
		if !ok {
			continue
		}
		if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
			continue
		}
		resultBasic, ok := sig.Results().At(0).Type().(*types.Basic)
		if ok && resultBasic.Kind() == types.String {
			return true
		}
	}
	return false
}
