package matcher

import (
	"testing"

	"github.com/seitarof/gen-dto/internal/parser"
)

func TestFieldMatcher_Match_CaseInsensitiveAndIgnore(t *testing.T) {
	src := &parser.StructInfo{
		Name: "User",
		Fields: []parser.FieldInfo{
			{Name: "ID"},
			{Name: "Name"},
			{Name: "Password"},
		},
	}
	dst := &parser.StructInfo{
		Name: "UserDTO",
		Fields: []parser.FieldInfo{
			{Name: "id"},
			{Name: "NAME"},
			{Name: "Email"},
		},
	}

	pairs := NewFieldMatcher().Match(src, dst, []string{"password"})
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	if pairs[0].SrcField.Name != "ID" || pairs[0].DstField.Name != "id" {
		t.Fatalf("unexpected first pair: %#v", pairs[0])
	}
}

func TestStructMatcher_MatchStructs_ExactName(t *testing.T) {
	srcInfos := []*parser.StructInfo{{Name: "Address"}, {Name: "User"}}
	dstInfos := []*parser.StructInfo{{Name: "User"}, {Name: "Tag"}}

	pairs := NewStructMatcher().MatchStructs(srcInfos, dstInfos)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Src.Name != "User" || pairs[0].Dst.Name != "User" {
		t.Fatalf("unexpected pair: %#v", pairs[0])
	}
}
