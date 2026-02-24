package resolver

import (
	"path"
	"strings"
	"unicode"

	"github.com/seitarof/gen-dto/internal/parser"
)

// ConversionPlan describes one field conversion.
type ConversionPlan struct {
	SrcField   parser.FieldInfo
	DstField   parser.FieldInfo
	Strategy   ConversionStrategy
	Expression string
}

// StructConversionPlan describes one struct converter function.
type StructConversionPlan struct {
	Src      *parser.StructInfo
	Dst      *parser.StructInfo
	FuncName string
	Plans    []ConversionPlan
}

// ConversionStrategy identifies conversion behavior.
type ConversionStrategy int

const (
	StrategyDirectAssign ConversionStrategy = iota
	StrategyBasicCast
	StrategyPointerWrap
	StrategyPointerUnwrap
	StrategyNullableToValue
	StrategyValueToNullable
	StrategyTimeToString
	StrategyStringToTime
	StrategySliceConvert
	StrategyNestedStruct
	StrategyNestedStructPtr
	StrategyNestedSlice
	StrategyCustomFunc
	StrategySkip
)

// DefaultConverterName returns generated converter function name.
func DefaultConverterName(srcPkgPath, srcName, dstPkgPath, dstName string) string {
	if srcName != dstName {
		return "Convert" + srcName + "To" + dstName
	}
	return "Convert" + packageToken(srcPkgPath) + srcName + "To" + packageToken(dstPkgPath) + dstName
}

func packageToken(pkgPath string) string {
	base := path.Base(strings.TrimSpace(pkgPath))
	if base == "" || base == "." || base == "/" {
		return "Pkg"
	}
	return toExportedToken(base)
}

func toExportedToken(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(parts) == 0 {
		return "Pkg"
	}

	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		first := runes[0]
		b.WriteRune(unicode.ToUpper(first))
		if len(runes) > 1 {
			b.WriteString(string(runes[1:]))
		}
	}
	if b.Len() == 0 {
		return "Pkg"
	}
	return b.String()
}
