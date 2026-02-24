package resolver

import (
	"go/types"
	"testing"

	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
)

func BenchmarkResolverResolve_MixedRules(b *testing.B) {
	r := New(DefaultRules()...)
	pairs, structPairs := benchmarkResolverInputs()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plans := r.Resolve(pairs, structPairs)
		if len(plans) != len(pairs) {
			b.Fatalf("unexpected plan count: got %d want %d", len(plans), len(pairs))
		}
	}
}

func benchmarkResolverInputs() ([]matcher.FieldPair, []matcher.StructPair) {
	timeType := newNamedStructField("CreatedAt", "CreatedAt", "time.Time", "time", "Time")
	srcAddress := newNamedStructField("Address", "Address", "source.Address", "example.com/source", "Address")
	dstAddress := newNamedStructField("Address", "Address", "dest.Address", "example.com/dest", "Address")
	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])

	pairs := []matcher.FieldPair{
		{
			SrcField: newBasicField("ID", "ID", "int", types.Typ[types.Int]),
			DstField: newBasicField("ID", "ID", "int64", types.Typ[types.Int64]),
		},
		{
			SrcField: newBasicField("Name", "Name", "string", types.Typ[types.String]),
			DstField: newPointerBasicField("Name", "Name", "string", types.Typ[types.String]),
		},
		{
			SrcField: newPointerBasicField("Alias", "Alias", "string", types.Typ[types.String]),
			DstField: newBasicField("Alias", "Alias", "string", types.Typ[types.String]),
		},
		{
			SrcField: newNamedStructField("Nick", "Nick", "sql.NullString", "database/sql", "NullString"),
			DstField: newBasicField("Nick", "Nick", "string", types.Typ[types.String]),
		},
		{
			SrcField: newBasicField("Label", "Label", "string", types.Typ[types.String]),
			DstField: newNamedStructField("Label", "Label", "sql.NullString", "database/sql", "NullString"),
		},
		{
			SrcField: timeType,
			DstField: newBasicField("CreatedAt", "CreatedAt", "string", types.Typ[types.String]),
		},
		{
			SrcField: newBasicField("UpdatedAt", "UpdatedAt", "string", types.Typ[types.String]),
			DstField: newNamedStructField("UpdatedAt", "UpdatedAt", "time.Time", "time", "Time"),
		},
		{
			SrcField: srcAddress,
			DstField: dstAddress,
		},
		{
			SrcField: newSliceBasicField("Scores", "Scores", "[]int", types.Typ[types.Int]),
			DstField: newSliceBasicField("Scores", "Scores", "[]int64", types.Typ[types.Int64]),
		},
		{
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
		},
	}

	structPairs := []matcher.StructPair{
		{
			Src: &parser.StructInfo{Name: "Address", PkgPath: "example.com/source"},
			Dst: &parser.StructInfo{Name: "Address", PkgPath: "example.com/dest"},
		},
	}

	return pairs, structPairs
}
