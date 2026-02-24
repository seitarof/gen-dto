package parser

import (
	"sort"
	"strings"

	"go/types"
)

type fieldCandidate struct {
	field     FieldInfo
	depth     int
	order     int
	ambiguous bool
}

func flattenFields(st *types.Struct, qualifier types.Qualifier) []FieldInfo {
	candidates := map[string]fieldCandidate{}
	order := 0
	collectFlattenedFields(st, nil, "", 0, qualifier, candidates, &order)

	sorted := make([]fieldCandidate, 0, len(candidates))
	for _, cand := range candidates {
		if cand.ambiguous {
			continue
		}
		sorted = append(sorted, cand)
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].order == sorted[j].order {
			return sorted[i].field.Name < sorted[j].field.Name
		}
		return sorted[i].order < sorted[j].order
	})

	fields := make([]FieldInfo, 0, len(sorted))
	for _, cand := range sorted {
		fields = append(fields, cand.field)
	}
	return fields
}

func collectFlattenedFields(
	st *types.Struct,
	prefix []string,
	embedFrom string,
	depth int,
	qualifier types.Qualifier,
	out map[string]fieldCandidate,
	order *int,
) {
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if f.Embedded() {
			embeddedStruct, embeddedName := resolveEmbeddedStruct(f.Type())
			if embeddedStruct == nil {
				continue
			}
			nextPrefix := appendPath(prefix, f.Name())
			nextEmbedFrom := embeddedName
			if nextEmbedFrom == "" {
				nextEmbedFrom = f.Name()
			}
			collectFlattenedFields(
				embeddedStruct,
				nextPrefix,
				nextEmbedFrom,
				depth+1,
				qualifier,
				out,
				order,
			)
			continue
		}
		if !f.Exported() {
			continue
		}

		field := FieldInfo{
			Name:       f.Name(),
			AccessPath: buildAccessPath(prefix, f.Name()),
			TypeStr:    types.TypeString(f.Type(), qualifier),
			TypeInfo:   analyzeType(f.Type()),
			Type:       f.Type(),
			IsExported: true,
			EmbedFrom:  embedFrom,
		}
		addCandidate(out, field, depth, order)
	}
}

func addCandidate(out map[string]fieldCandidate, field FieldInfo, depth int, order *int) {
	key := strings.ToLower(field.Name)
	cand, exists := out[key]
	if !exists {
		out[key] = fieldCandidate{field: field, depth: depth, order: *order}
		*order = *order + 1
		return
	}

	if depth < cand.depth {
		out[key] = fieldCandidate{field: field, depth: depth, order: *order}
		*order = *order + 1
		return
	}
	if depth > cand.depth {
		return
	}

	if cand.field.AccessPath != field.AccessPath {
		cand.ambiguous = true
		out[key] = cand
	}
}

func appendPath(prefix []string, part string) []string {
	next := make([]string, 0, len(prefix)+1)
	next = append(next, prefix...)
	next = append(next, part)
	return next
}

func buildAccessPath(prefix []string, fieldName string) string {
	if len(prefix) == 0 {
		return fieldName
	}
	parts := appendPath(prefix, fieldName)
	return strings.Join(parts, ".")
}

func resolveEmbeddedStruct(t types.Type) (*types.Struct, string) {
	switch v := t.(type) {
	case *types.Alias:
		return resolveEmbeddedStruct(v.Rhs())
	case *types.Named:
		if st, ok := v.Underlying().(*types.Struct); ok {
			return st, v.Obj().Name()
		}
	case *types.Pointer:
		return resolveEmbeddedStruct(v.Elem())
	}
	return nil, ""
}
