package parser

import (
	"fmt"
	"log"
	"strings"

	"go/types"

	"golang.org/x/tools/go/packages"
)

// Parser extracts struct metadata from Go packages.
type Parser interface {
	Parse(pkgPath string, typeName string) (*StructInfo, error)
	ParseRecursive(pkgPath string, typeName string) ([]*StructInfo, error)
}

type parserImpl struct{}

// New returns default parser.
func New() Parser {
	return &parserImpl{}
}

func (p *parserImpl) Parse(pkgPath string, typeName string) (*StructInfo, error) {
	cache := map[string]*packages.Package{}
	return p.parseWithCache(pkgPath, typeName, cache)
}

func (p *parserImpl) parseWithCache(
	pkgPath string,
	typeName string,
	cache map[string]*packages.Package,
) (*StructInfo, error) {
	pkg, err := p.loadPackage(pkgPath, cache)
	if err != nil {
		return nil, err
	}

	if pkg.Types == nil || pkg.Types.Scope() == nil {
		return nil, fmt.Errorf("type info unavailable for package %q", pkgPath)
	}

	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return nil, fmt.Errorf("struct %q not found in package %q", typeName, pkgPath)
	}

	st, ok := extractStructType(obj.Type())
	if !ok {
		return nil, fmt.Errorf("%q in package %q is not a struct type", typeName, pkgPath)
	}

	qualifier := func(p *types.Package) string {
		if p == nil {
			return ""
		}
		if pkg.Types != nil && p.Path() == pkg.Types.Path() {
			return ""
		}
		return p.Name()
	}

	return &StructInfo{
		Name:    typeName,
		PkgPath: pkg.Types.Path(),
		PkgName: pkg.Name,
		Fields:  flattenFields(st, qualifier),
	}, nil
}

func (p *parserImpl) loadPackage(pkgPath string, cache map[string]*packages.Package) (*packages.Package, error) {
	if cached, ok := cache[pkgPath]; ok {
		return cached, nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedModule,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("load package %q: %w", pkgPath, err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("package %q has compilation errors", pkgPath)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("package %q not found", pkgPath)
	}
	cache[pkgPath] = pkgs[0]
	return pkgs[0], nil
}

func (p *parserImpl) ParseRecursive(pkgPath string, typeName string) ([]*StructInfo, error) {
	visited := map[string]bool{}
	cache := map[string]*packages.Package{}
	rootPkg, err := p.loadPackage(pkgPath, cache)
	if err != nil {
		return nil, err
	}

	rootModulePath := ""
	if rootPkg.Module != nil {
		rootModulePath = rootPkg.Module.Path
	}

	result := []*StructInfo{}
	if err := p.parseRec(pkgPath, typeName, visited, cache, rootModulePath, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *parserImpl) parseRec(
	pkgPath string,
	typeName string,
	visited map[string]bool,
	cache map[string]*packages.Package,
	rootModulePath string,
	result *[]*StructInfo,
) error {
	info, err := p.parseWithCache(pkgPath, typeName, cache)
	if err != nil {
		return err
	}

	key := info.PkgPath + "." + info.Name
	if visited[key] {
		return nil
	}
	visited[key] = true

	for _, f := range info.Fields {
		nestedPkg, nestedName, ok := nestedStructRef(f.TypeInfo)
		if !ok {
			continue
		}
		if !shouldRecurseNestedPackage(nestedPkg, info.PkgPath, rootModulePath) {
			continue
		}
		if visited[nestedPkg+"."+nestedName] {
			continue
		}
		if err := p.parseRec(nestedPkg, nestedName, visited, cache, rootModulePath, result); err != nil {
			log.Printf("gen-dto: warning: nested struct %q not found, skipped", nestedName)
			continue
		}
	}

	*result = append(*result, info)
	return nil
}

func extractStructType(t types.Type) (*types.Struct, bool) {
	switch v := t.(type) {
	case *types.Alias:
		return extractStructType(v.Rhs())
	case *types.Named:
		return extractStructType(v.Underlying())
	case *types.Struct:
		return v, true
	default:
		return nil, false
	}
}

func nestedStructRef(detail TypeDetail) (pkgPath string, structName string, ok bool) {
	switch detail.Kind {
	case TypeKindStruct:
		if detail.PkgPath == "" || detail.StructName == "" {
			return "", "", false
		}
		return detail.PkgPath, detail.StructName, true
	case TypeKindPointer, TypeKindSlice:
		if detail.ElemType == nil {
			return "", "", false
		}
		return nestedStructRef(*detail.ElemType)
	default:
		return "", "", false
	}
}

func shouldRecurseNestedPackage(nestedPkgPath, currentPkgPath, rootModulePath string) bool {
	if nestedPkgPath == "" {
		return false
	}
	if nestedPkgPath == currentPkgPath {
		return true
	}
	if rootModulePath == "" {
		return false
	}
	return nestedPkgPath == rootModulePath || strings.HasPrefix(nestedPkgPath, rootModulePath+"/")
}

func analyzeType(t types.Type) TypeDetail {
	switch v := t.(type) {
	case *types.Alias:
		return analyzeType(v.Rhs())
	case *types.Basic:
		return TypeDetail{
			Kind:      TypeKindBasic,
			IsBasic:   true,
			BasicKind: v.Name(),
			TypeName:  v.Name(),
		}
	case *types.Pointer:
		elem := analyzeType(v.Elem())
		return TypeDetail{
			Kind:     TypeKindPointer,
			ElemType: &elem,
			TypeName: "*" + elem.TypeName,
		}
	case *types.Slice:
		elem := analyzeType(v.Elem())
		return TypeDetail{
			Kind:     TypeKindSlice,
			ElemType: &elem,
			TypeName: "[]" + elem.TypeName,
		}
	case *types.Map:
		key := analyzeType(v.Key())
		elem := analyzeType(v.Elem())
		return TypeDetail{
			Kind:     TypeKindMap,
			KeyType:  &key,
			ElemType: &elem,
			TypeName: "map[" + key.TypeName + "]" + elem.TypeName,
		}
	case *types.Interface:
		return TypeDetail{Kind: TypeKindInterface, TypeName: "interface"}
	case *types.Named:
		obj := v.Obj()
		pkgPath := ""
		typeName := obj.Name()
		if obj.Pkg() != nil {
			pkgPath = obj.Pkg().Path()
			typeName = obj.Pkg().Path() + "." + obj.Name()
		}

		switch under := v.Underlying().(type) {
		case *types.Struct:
			return TypeDetail{
				Kind:       TypeKindStruct,
				PkgPath:    pkgPath,
				StructName: obj.Name(),
				TypeName:   typeName,
			}
		case *types.Basic:
			return TypeDetail{
				Kind:      TypeKindBasic,
				PkgPath:   pkgPath,
				IsBasic:   true,
				BasicKind: under.Name(),
				TypeName:  typeName,
			}
		case *types.Pointer:
			elem := analyzeType(under.Elem())
			return TypeDetail{Kind: TypeKindPointer, ElemType: &elem, PkgPath: pkgPath, TypeName: typeName}
		case *types.Slice:
			elem := analyzeType(under.Elem())
			return TypeDetail{Kind: TypeKindSlice, ElemType: &elem, PkgPath: pkgPath, TypeName: typeName}
		case *types.Map:
			key := analyzeType(under.Key())
			elem := analyzeType(under.Elem())
			return TypeDetail{Kind: TypeKindMap, KeyType: &key, ElemType: &elem, PkgPath: pkgPath, TypeName: typeName}
		case *types.Interface:
			return TypeDetail{Kind: TypeKindInterface, PkgPath: pkgPath, TypeName: typeName}
		default:
			return TypeDetail{Kind: TypeKindOther, PkgPath: pkgPath, TypeName: typeName}
		}
	default:
		return TypeDetail{Kind: TypeKindOther, TypeName: strings.TrimSpace(types.TypeString(t, nil))}
	}
}
