package generator

type SwaggerSpec struct {
	Swagger  string                         `json:"swagger"`
	Info     Info                           `json:"info"`
	Host     string                         `json:"host"`
	BasePath string                         `json:"basePath"`
	Schemes  []string                       `json:"schemes"`
	Consumes []string                       `json:"consumes"`
	Produces []string                       `json:"produces"`
	Paths    map[string]map[string]PathItem `json:"paths"`
}

type Info struct {
	Version        string  `json:"version"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	TermsOfService string  `json:"termsOfService"`
	Contact        Contact `json:"contact"`
	License        License `json:"license"`
}

type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type PathItem struct {
	Description string              `json:"description"`
	OperationId string              `json:"operationId"`
	Parameters  []Parameter         `json:"parameters"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name             string `json:"name"`
	In               string `json:"in"`
	Description      string `json:"description"`
	Required         bool   `json:"required"`
	Type             string `json:"type"`
	Format           string `json:"format,omitempty"`
	CollectionFormat string `json:"collectionFormat,omitempty"`
	Items            *Items `json:"items,omitempty"`
}

type Items struct {
	Type string `json:"type"`
}

type Response struct {
	Description string `json:"description"`
	Schema      Schema `json:"schema"`
}

type Schema struct {
	Type        string             `json:"type"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
}
