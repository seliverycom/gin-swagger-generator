package collector

import (
	"fmt"
	"github.com/seliverycom/gin-swagger-generator/config"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"strings"
	"unicode"
)

type EndpointData struct {
	FunctionName string
	Method       string
	Endpoint     string
	Request      ReqResp
	Response     ReqResp
}

type ReqResp struct {
	Name   string
	Fields []ReqRespField
}

type ReqRespField struct {
	Name    string
	VarName string
	Type    string
}

var Constants = make(map[string]interface{})
var structs = make(map[string]ReqResp)
var endpoints = make(map[string]EndpointData)

type Service struct {
	conf          config.Config
	activePackage Package
}

type Package struct {
	Name      string
	Endpoints map[string]EndpointData
}

func New(conf config.Config) *Service {
	return &Service{
		conf: conf,
	}
}
func (s *Service) Collect(filePath string) {
	// Create a new token file set
	fset := token.NewFileSet()

	// Parse the file
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		ts, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		s.collectConst(ts)
		return true
	})

	// Inspect the AST and print the names of all functions
	ast.Inspect(node, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		s.collectStruct(ts)
		return true
	})

	// Inspect the AST and print the names of all functions
	ast.Inspect(node, func(n ast.Node) bool {
		// Check if the node is a function declaration
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true // not a function declaration, continue
		}

		if fn.Name.String() == "New" {
			return true
		}

		s.collectEndpoints(fn)
		return true
	})

	s.activePackage = Package{
		Name:      node.Name.String(),
		Endpoints: endpoints,
	}
}

func (s *Service) collectEndpoints(fn *ast.FuncDecl) {
	meta := EndpointData{
		FunctionName: fn.Name.String(),
	}

	// Parsing Method comments
	comment := fn.Doc.Text()
	commentByLine := strings.Split(comment, "\n")
	for _, line := range commentByLine {
		if strings.HasPrefix(line, "@endpoint") {
			meta.Endpoint = strings.TrimSpace(strings.TrimPrefix(line, "@endpoint"))
		} else if strings.HasPrefix(line, "@method") {
			meta.Method = strings.TrimSpace(strings.TrimPrefix(line, "@method"))
		}
	}

	// Parse Request
	reqType := types.ExprString(fn.Type.Params.List[1].Type)
	request, ok := structs[reqType]

	if ok {
		meta.Request = request
	}

	// Parse response
	respType := types.ExprString(fn.Type.Results.List[0].Type)
	response, ok := structs[respType]
	if ok {
		meta.Response = response
	}

	endpoints[meta.FunctionName] = meta
}

func (s *Service) collectConst(spec *ast.ValueSpec) {
	if len(spec.Values) > 0 {
		for i, ident := range spec.Names {
			name := ident.Name
			if len(spec.Values) > 0 {
				if basicLit, ok := spec.Values[i].(*ast.BasicLit); ok {
					value := basicLit.Value
					Constants[name] = value
				}
			}
		}
	}
}

func (s *Service) collectStruct(spec *ast.TypeSpec) {
	st, ok := spec.Type.(*ast.StructType)

	if ok {
		item := ReqResp{
			Name: spec.Name.String(),
		}

		var fields = make([]ReqRespField, 0)
		for _, field := range st.Fields.List {
			//tagStr := field.Tag.Value
			//tagStr = strings.Trim(tagStr, "`")
			//tags, err := structtag.Parse(tagStr)
			//if err != nil {
			//	panic(err)
			//}
			//x, _ := json.Marshal(field.Type.(*ast.ArrayType))
			//fmt.Println("type:", string(x), reflect.TypeOf(field.Type))
			fmt.Printf("Field: %s\n", field.Names[0].Name)
			fmt.Println("expr:", s.exprToString(field.Type))
			_field := ReqRespField{
				Name:    field.Names[0].Name,
				VarName: s.camelCaseToUnderscore(field.Names[0].Name),
				Type:    s.exprToString(field.Type),
			}
			fields = append(fields, _field)
		}

		item.Fields = fields

		structs[item.Name] = item
	}
}

func (s *Service) camelCaseToUnderscore(input string) string {
	var result strings.Builder
	for i, r := range input {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func (s *Service) GetActivePackage() Package {
	return s.activePackage
}

func (s *Service) exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return s.exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + s.exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + s.exprToString(t.Elt)
	// Add more cases for other types as needed
	default:
		return fmt.Sprintf("%v", expr)
	}
}

func (s *Service) GetStruct(name string) *ReqResp {
	myStruct, ok := structs[name]

	if !ok {
		return nil
	}

	return &myStruct
}

func GetAllEndpoints() map[string]EndpointData {
	return endpoints
}
