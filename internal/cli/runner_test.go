package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/seitarof/gen-dto/internal/generator"
	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
	"github.com/seitarof/gen-dto/internal/resolver"
)

func TestRunner_Run_IncludesRootPairAndCustomFuncName(t *testing.T) {
	srcUser := &parser.StructInfo{Name: "User", PkgPath: "example.com/src", PkgName: "model"}
	srcAddress := &parser.StructInfo{Name: "Address", PkgPath: "example.com/src", PkgName: "model"}
	dstUser := &parser.StructInfo{Name: "UserResponse", PkgPath: "example.com/dst", PkgName: "dto"}
	dstAddress := &parser.StructInfo{Name: "AddressResponse", PkgPath: "example.com/dst", PkgName: "dto"}

	p := &mockParser{
		srcInfos: []*parser.StructInfo{srcAddress, srcUser},
		dstInfos: []*parser.StructInfo{dstAddress, dstUser},
	}
	sm := &mockStructMatcher{
		pairs: []matcher.StructPair{{Src: srcAddress, Dst: dstAddress}},
	}
	fm := &mockFieldMatcher{}
	rv := &mockResolver{}
	gen := &mockGenerator{}

	r := NewRunner(p, sm, fm, rv, gen)
	cfg := &Config{
		SrcType:      "User",
		SrcPath:      "src/path",
		DstType:      "UserResponse",
		DstPath:      "dst/path",
		Filename:     "generated.go",
		FuncName:     "BuildUserResponse",
		IgnoreFields: []string{"Password"},
	}

	if err := r.Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if gen.callCount != 1 {
		t.Fatalf("generator call count = %d, want 1", gen.callCount)
	}
	if len(gen.plans) != 4 {
		t.Fatalf("generated plans = %d, want 4", len(gen.plans))
	}

	forwardRoot := gen.plans[0]
	if forwardRoot.Src.Name != "User" || forwardRoot.Dst.Name != "UserResponse" {
		t.Fatalf("first plan should be forward root pair, got %s -> %s", forwardRoot.Src.Name, forwardRoot.Dst.Name)
	}
	if forwardRoot.FuncName != "BuildUserResponse" {
		t.Fatalf("root func name = %s, want BuildUserResponse", forwardRoot.FuncName)
	}

	forwardNested := gen.plans[1]
	if forwardNested.FuncName != "ConvertAddressToAddressResponse" {
		t.Fatalf("nested func name = %s, want ConvertAddressToAddressResponse", forwardNested.FuncName)
	}

	reverseRoot := gen.plans[2]
	if reverseRoot.FuncName != "ConvertUserResponseToUser" {
		t.Fatalf("reverse root func name = %s, want ConvertUserResponseToUser", reverseRoot.FuncName)
	}

	reverseNested := gen.plans[3]
	if reverseNested.FuncName != "ConvertAddressResponseToAddress" {
		t.Fatalf("reverse nested func name = %s, want ConvertAddressResponseToAddress", reverseNested.FuncName)
	}

	if fm.callCount != 4 {
		t.Fatalf("field matcher call count = %d, want 4", fm.callCount)
	}
	if len(fm.lastIgnoreFields) != 1 || fm.lastIgnoreFields[0] != "Password" {
		t.Fatalf("ignore fields not forwarded: %#v", fm.lastIgnoreFields)
	}
}

func TestRunner_Run_ParseSrcError(t *testing.T) {
	r := NewRunner(
		&mockParser{srcErr: errors.New("src failed")},
		&mockStructMatcher{},
		&mockFieldMatcher{},
		&mockResolver{},
		&mockGenerator{},
	)

	err := r.Run(&Config{SrcType: "User", SrcPath: "src", DstType: "User", DstPath: "dst", Filename: "out.go"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse src") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunner_Run_NoMatchingStructs(t *testing.T) {
	r := NewRunner(
		&mockParser{
			srcInfos: []*parser.StructInfo{{Name: "User", PkgPath: "example.com/src"}},
			dstInfos: []*parser.StructInfo{{Name: "Other", PkgPath: "example.com/dst"}},
		},
		&mockStructMatcher{pairs: nil},
		&mockFieldMatcher{},
		&mockResolver{},
		&mockGenerator{},
	)

	err := r.Run(&Config{SrcType: "User", SrcPath: "src", DstType: "UserResponse", DstPath: "dst", Filename: "out.go"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no matching structs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReverseStructPairs_SkipsSameTypePair(t *testing.T) {
	user := &parser.StructInfo{Name: "User", PkgPath: "example.com/model"}
	forward := []matcher.StructPair{
		{Src: user, Dst: user},
	}

	reverse := reverseStructPairs(forward)
	if len(reverse) != 0 {
		t.Fatalf("expected no reverse pairs, got %d", len(reverse))
	}
}

type mockParser struct {
	srcInfos []*parser.StructInfo
	dstInfos []*parser.StructInfo
	srcErr   error
	dstErr   error
	calls    int
}

func (m *mockParser) Parse(pkgPath string, typeName string) (*parser.StructInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *mockParser) ParseRecursive(pkgPath string, typeName string) ([]*parser.StructInfo, error) {
	if m.calls == 0 {
		m.calls++
		if m.srcErr != nil {
			return nil, m.srcErr
		}
		return m.srcInfos, nil
	}
	if m.dstErr != nil {
		return nil, m.dstErr
	}
	return m.dstInfos, nil
}

type mockStructMatcher struct {
	pairs []matcher.StructPair
}

func (m *mockStructMatcher) MatchStructs(srcInfos, dstInfos []*parser.StructInfo) []matcher.StructPair {
	return m.pairs
}

type mockFieldMatcher struct {
	callCount        int
	lastIgnoreFields []string
}

func (m *mockFieldMatcher) Match(src, dst *parser.StructInfo, ignoreFields []string) []matcher.FieldPair {
	m.callCount++
	m.lastIgnoreFields = append([]string(nil), ignoreFields...)
	return []matcher.FieldPair{{
		SrcField: parser.FieldInfo{Name: "Name", AccessPath: "Name", TypeStr: "string"},
		DstField: parser.FieldInfo{Name: "Name", AccessPath: "Name", TypeStr: "string"},
	}}
}

type mockResolver struct{}

func (m *mockResolver) Resolve(pairs []matcher.FieldPair, structPairs []matcher.StructPair) []resolver.ConversionPlan {
	if len(pairs) == 0 {
		return nil
	}
	return []resolver.ConversionPlan{{
		SrcField:   pairs[0].SrcField,
		DstField:   pairs[0].DstField,
		Strategy:   resolver.StrategyDirectAssign,
		Expression: "dst.Name = src.Name",
	}}
}

type mockGenerator struct {
	callCount int
	cfg       generator.Config
	plans     []resolver.StructConversionPlan
	err       error
}

func (m *mockGenerator) Generate(cfg generator.Config, plans []resolver.StructConversionPlan) error {
	m.callCount++
	m.cfg = cfg
	m.plans = append([]resolver.StructConversionPlan(nil), plans...)
	return m.err
}
