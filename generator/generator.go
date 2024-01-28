package generator

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/seliverycom/gin-swagger-generator/collector"
	"github.com/seliverycom/gin-swagger-generator/config"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Service struct {
	conf   config.Config
	folder embed.FS
}

func New(conf config.Config, folder embed.FS) *Service {
	return &Service{
		conf:   conf,
		folder: folder,
	}
}

func (s *Service) Init() {
	path := s.conf.ApiPath
	isFile, err := s.isFilePath(path)
	if err != nil {
		panic(err)
	}

	collection := collector.New(s.conf)

	if isFile {
		s.GenerateForFile(path, collection)
	} else {
		s.generateForAllFilesInFolder(path, collection)
	}

	s.generateSwagger(collection)

	err = s.copyDir(s.folder, "static/swagger-ui")

	if err != nil {
		panic(err)
	}

}

func (s *Service) generateForAllFilesInFolder(path string, collection *collector.Service) {
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// if it's file - try to generate for it
		if !d.IsDir() {
			s.GenerateForFile(path, collection)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func (s *Service) GenerateForFile(path string, collection *collector.Service) {
	// check if we need to skip this file
	if s.checkFileSkips(path) {
		return
	}

	collection.Collect(path)

	s.generateServer(collection, path)
}

func (s *Service) generateServer(collection *collector.Service, path string) {
	server := s.getTemplate("server")

	var endpointList string
	for _, endpoint := range collection.GetActivePackage().Endpoints {
		ginTemplate := s.getTemplate("gin")
		ginTemplate = strings.Replace(ginTemplate, "{{ENDPOINT}}", endpoint.Endpoint, -1)
		ginTemplate = strings.Replace(ginTemplate, "{{METHOD}}", endpoint.Method, -1)
		ginTemplate = strings.Replace(ginTemplate, "{{FUNCTION}}", endpoint.FunctionName, -1)
		ginTemplate = strings.Replace(ginTemplate, "{{REQUEST}}", s.generateRequest(endpoint), -1)

		endpointList += ginTemplate + "\n"
	}

	server = strings.Replace(server, "{{ENDPOINTS}}", endpointList, -1)
	server = strings.Replace(server, "{{PACKAGE}}", collection.GetActivePackage().Name, -1)

	if val, ok := collector.Constants["mainServicePackagePath"]; ok {
		if str, ok := val.(string); ok {
			server = strings.Replace(server, "{{MAIN_SERVICE_PACKAGE_PATH}}", str, -1)
		}
	}

	server, err := s.gofmt(server)

	if err != nil {
		panic(err)
	}

	handlerDir := filepath.Dir(path)
	s.checkAndCreateFolder(handlerDir)
	err = os.WriteFile(handlerDir+"/"+s.conf.GeneratedFileName, []byte(server), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("File created: " + handlerDir + "/" + s.conf.GeneratedFileName)
}

func (s *Service) checkAndCreateFolder(folder string) {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		// Folder does not exist, so create it
		err := os.Mkdir(folder, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}

// Generating request params mapping to Request struct
func (s *Service) generateRequest(endpoint collector.EndpointData) (request string) {
	if endpoint.Method == "GET" {
		request = s.getTemplate("get-request")
	} else {
		request = s.getTemplate("post-request")
	}
	request = strings.Replace(request, "{{REQUEST_NAME}}", endpoint.Request.Name, -1)

	return request
}

func (s *Service) getTemplate(template string) string {
	var file string
	switch template {
	case "get-request":
		file = "static/inc/get_request.go.tmpl"
		break
	case "post-request":
		file = "static/inc/post_request.go.tmpl"
		break
	case "gin":
		file = "static/inc/gin.go.tmpl"
		break
	case "server":
		file = "static/inc/server.go.tmpl"
		break
	}

	b, err := s.folder.ReadFile(file)
	if err != nil {
		panic(err)
	}

	return string(b)
}

// check if we have path to file or dir
func (s *Service) isFilePath(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		// do directory stuff
		return false, nil
	case mode.IsRegular():
		// do file stuff
		return true, nil
	default:
		return true, nil
	}
}

// Check do we need to skip this file or not
func (s *Service) checkFileSkips(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "// @sparkle-generated") {
			return true
		}
	}
	return false
}

func (s *Service) gofmt(source string) (string, error) {
	cmd := exec.Command("gofmt")

	// Set the input to the source code
	cmd.Stdin = bytes.NewReader([]byte(source))

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func (s *Service) generateSwagger(collection *collector.Service) {
	swagger := SwaggerSpec{
		Swagger:  "2.0",
		Schemes:  []string{"https"},
		Consumes: []string{"application/json"},
		Produces: []string{"application/json"},
	}
	swagger.Paths = make(map[string]map[string]PathItem)

	for _, endpoint := range collector.GetAllEndpoints() {
		params := make([]Parameter, 0)

		for _, param := range endpoint.Request.Fields {
			params = append(params, Parameter{
				Name:        param.VarName,
				In:          "query",
				Description: param.Name,
				Type:        s.simpleTypeToScheme(param.Type).Type,
			})
		}

		successResponse := make(map[string]*Schema)
		for _, resp := range endpoint.Response.Fields {
			successResponse[resp.VarName] = s.getScheme(resp.Type, collection)
		}

		responses := make(map[string]Response)
		responses["200"] = Response{
			Description: fmt.Sprintf("Success response for %s", endpoint.Endpoint),
			Schema: Schema{
				Type:       "object",
				Properties: successResponse,
			},
		}

		if swagger.Paths[endpoint.Endpoint] == nil {
			swagger.Paths[endpoint.Endpoint] = make(map[string]PathItem)
		}

		swagger.Paths[endpoint.Endpoint][strings.ToLower(endpoint.Method)] = PathItem{
			Description: "",
			OperationId: endpoint.Endpoint,
			Parameters:  params,
			Responses:   responses,
		}
	}

	s.checkAndCreateFolder(s.conf.SwaggerPath)
	marshaled, err := json.Marshal(swagger)

	if err != nil {
		panic(err)
	}

	err = os.WriteFile(fmt.Sprintf("%s/doc.json", s.conf.SwaggerPath), marshaled, 0644)
	if err != nil {
		panic(err)
	}
}

func (s *Service) getScheme(fieldType string, collection *collector.Service) *Schema {

	if strings.HasPrefix(fieldType, "[]") {
		arrayItem := strings.TrimLeft(fieldType, "[]")

		itemField := collection.GetStruct(arrayItem)

		if itemField != nil {
			properties := make(map[string]*Schema)
			for _, inheritedField := range itemField.Fields {
				sc := s.getScheme(inheritedField.Type, collection)
				properties[inheritedField.VarName] = sc
			}

			return &Schema{
				Type: "array",
				Items: &Schema{
					Type:       "object",
					Properties: properties,
				},
			}
		}
		return &Schema{
			Type:  "array",
			Items: s.simpleTypeToScheme(arrayItem),
		}
	}

	return s.simpleTypeToScheme(fieldType)

}

func (s *Service) simpleTypeToScheme(fieldType string) *Schema {
	switch fieldType {
	case "string":
		return &Schema{Type: "string"}
	case "int", "int32", "int64":
		return &Schema{Type: "integer", Format: fieldType}
	case "float", "float32", "float64":
		return &Schema{Type: "number", Format: fieldType}
	default:
		return &Schema{Type: fieldType}
	}
}

func (s *Service) copyDir(sourceFS fs.FS, source string) error {
	return fs.WalkDir(sourceFS, source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create directories
		if d.IsDir() {
			return os.MkdirAll(path, os.ModePerm)
		}

		// Copy files
		file, err := sourceFS.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		newFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer newFile.Close()

		_, err = io.Copy(newFile, file)
		return err
	})
}
