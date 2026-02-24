package parser

import (
	"strings"
	"testing"
)

func TestParse_BasicStruct(t *testing.T) {
	p := New()

	info, err := p.Parse("github.com/seitarof/gen-dto/testdata/parserbasic", "User")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if info.Name != "User" {
		t.Fatalf("expected Name=User, got %s", info.Name)
	}

	if fieldByName(info.Fields, "hidden") != nil {
		t.Fatal("unexported field should be excluded")
	}

	profile := fieldByName(info.Fields, "Profile")
	if profile == nil {
		t.Fatal("Profile field not found")
	}
	if profile.TypeInfo.Kind != TypeKindStruct {
		t.Fatalf("Profile kind = %v, want TypeKindStruct", profile.TypeInfo.Kind)
	}

	ptr := fieldByName(info.Fields, "Ptr")
	if ptr == nil || ptr.TypeInfo.Kind != TypeKindPointer {
		t.Fatalf("Ptr should be pointer field, got %#v", ptr)
	}

	tags := fieldByName(info.Fields, "Tags")
	if tags == nil || tags.TypeInfo.Kind != TypeKindSlice {
		t.Fatalf("Tags should be slice field, got %#v", tags)
	}
}

func TestParseRecursive_NestedAndCycle(t *testing.T) {
	p := New()

	infos, err := p.ParseRecursive("github.com/seitarof/gen-dto/testdata/parsernested", "Root")
	if err != nil {
		t.Fatalf("ParseRecursive() error = %v", err)
	}

	if len(infos) != 3 {
		t.Fatalf("expected 3 structs, got %d", len(infos))
	}

	wantOrder := []string{"Leaf", "Child", "Root"}
	for i, want := range wantOrder {
		if infos[i].Name != want {
			t.Fatalf("order[%d] = %s, want %s", i, infos[i].Name, want)
		}
	}
}

func TestParse_EmbeddedAndConflict(t *testing.T) {
	p := New()

	info, err := p.Parse("github.com/seitarof/gen-dto/testdata/parserembed", "User")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	name := fieldByName(info.Fields, "Name")
	if name == nil {
		t.Fatal("Name field not found")
	}
	if name.AccessPath != "Name" {
		t.Fatalf("Name access path = %q, want direct field", name.AccessPath)
	}

	id := fieldByName(info.Fields, "ID")
	if id == nil {
		t.Fatal("ID field not found")
	}
	if id.AccessPath != "Base.ID" {
		t.Fatalf("ID access path = %q, want Base.ID", id.AccessPath)
	}

	if fieldByName(info.Fields, "Code") != nil {
		t.Fatal("Code should be dropped due to same-depth embedded conflict")
	}
}

func TestParse_TypeNotFound(t *testing.T) {
	p := New()

	_, err := p.Parse("github.com/seitarof/gen-dto/testdata/parserbasic", "NotExist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParse_TypeAliasToStruct(t *testing.T) {
	p := New()

	info, err := p.Parse("github.com/seitarof/gen-dto/testdata/aliasdst", "PatientProviderType")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if info.Name != "PatientProviderType" {
		t.Fatalf("expected Name=PatientProviderType, got %s", info.Name)
	}

	if fieldByName(info.Fields, "ProviderCode") == nil {
		t.Fatal("ProviderCode field not found in aliased struct")
	}
}

func TestParse_FieldWithTypeAlias(t *testing.T) {
	p := New()

	info, err := p.Parse("github.com/seitarof/gen-dto/testdata/aliasdst", "PatientDTO")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	field := fieldByName(info.Fields, "ProviderType")
	if field == nil {
		t.Fatal("ProviderType field not found")
	}
	if field.TypeInfo.Kind != TypeKindStruct {
		t.Fatalf("ProviderType kind = %v, want TypeKindStruct", field.TypeInfo.Kind)
	}
	if field.TypeInfo.StructName != "GetPatientProviderTypeResponse" {
		t.Fatalf("unexpected StructName: %s", field.TypeInfo.StructName)
	}
}

func TestShouldRecurseNestedPackage(t *testing.T) {
	tests := []struct {
		name       string
		nestedPkg  string
		currentPkg string
		modulePath string
		want       bool
	}{
		{
			name:       "same package",
			nestedPkg:  "example.com/mod/a",
			currentPkg: "example.com/mod/a",
			modulePath: "example.com/mod",
			want:       true,
		},
		{
			name:       "same module different package",
			nestedPkg:  "example.com/mod/b",
			currentPkg: "example.com/mod/a",
			modulePath: "example.com/mod",
			want:       true,
		},
		{
			name:       "outside module",
			nestedPkg:  "time",
			currentPkg: "example.com/mod/a",
			modulePath: "example.com/mod",
			want:       false,
		},
		{
			name:       "empty module path only same package allowed",
			nestedPkg:  "example.com/other",
			currentPkg: "example.com/mod/a",
			modulePath: "",
			want:       false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRecurseNestedPackage(tc.nestedPkg, tc.currentPkg, tc.modulePath)
			if got != tc.want {
				t.Fatalf("shouldRecurseNestedPackage() = %v, want %v", got, tc.want)
			}
		})
	}
}

func fieldByName(fields []FieldInfo, name string) *FieldInfo {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}
