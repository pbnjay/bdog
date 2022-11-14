package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type OpenAPI struct {
	OpenAPIVersion string              `json:"openapi"`
	Info           APIInfo             `json:"info"`
	Servers        []APIServer         `json:"servers"`
	Paths          map[string]*APIPath `json:"paths"`
}

type APIInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type APIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type APIPath struct {
	Get    *APIOperation `json:"get,omitempty"`
	Patch  *APIOperation `json:"patch,omitempty"`
	Post   *APIOperation `json:"post,omitempty"`
	Delete *APIOperation `json:"delete,omitempty"`
}

type APIOperation struct {
	Summary    string                 `json:"summary"`
	Parameters []APIParameter         `json:"parameters,omitempty"`
	Responses  map[string]APIResponse `json:"responses"`
}

type APIParameter struct {
	Name        string        `json:"name"`
	In          string        `json:"in"` // "query", "header", "path" or "cookie"
	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required"`
	Schema      APISchemaType `json:"schema"`
}

type APISchemaType struct {
	Type string `json:"type"`
}

type APIResponse struct {
	Description string                    `json:"description"`
	Content     map[string]APIContentType `json:"content,omitempty"`
}

type APIContentType struct {
	Schema  string      `json:"schema,omitempty"`
	Example interface{} `json:"example,omitempty"`
}

func (s *OpenAPI) NewHandler(method, path string) *APIOperation {
	defaultResponseCode := "200"
	switch strings.ToUpper(method) {
	case "PATCH", "DELETE":
		defaultResponseCode = "204"
	case "POST":
		defaultResponseCode = "201"
	}
	newOp := &APIOperation{
		Responses: map[string]APIResponse{
			defaultResponseCode: {
				Description: "A successfull response",
			},
		},
	}

	if s.Paths == nil {
		s.Paths = make(map[string]*APIPath)
	}
	if strings.Contains(path, ":") {
		parts := strings.Split(path, "/")
		for i, p := range parts {
			if p == "" {
				continue
			}
			if p[:1] == ":" {
				parts[i] = "{" + p[1:] + "}"
				newOp.Parameters = append(newOp.Parameters, APIParameter{
					Name:     p[1:],
					In:       "path",
					Required: true,
					Schema:   APISchemaType{Type: "string"},
				})
			}
		}
		path = strings.Join(parts, "/")
	}

	_, exists := s.Paths[path]
	if !exists {
		s.Paths[path] = &APIPath{}
	}

	switch strings.ToLower(method) {
	case "get":
		s.Paths[path].Get = newOp
	case "patch":
		s.Paths[path].Patch = newOp
	case "post":
		s.Paths[path].Post = newOp
	case "delete":
		s.Paths[path].Delete = newOp
	default:
		panic("method type not supported (only get,patch,post,delete)")
	}
	return newOp
}

func (s *OpenAPI) Handler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		json.NewEncoder(w).Encode(s)
	}
}

func NewOpenAPISpec(apiName, apiVersion, listenURL string) *OpenAPI {
	if apiName == "" {
		apiName = "Unnamed bdog API"
	}
	if apiVersion == "" {
		apiVersion = "0.0.01"
	}
	if listenURL == "" {
		listenURL = "http://127.0.0.1:8080/"
	}
	return &OpenAPI{
		OpenAPIVersion: "3.0.3",
		Info: APIInfo{
			Title:   apiName,
			Version: apiVersion,
		},
		Servers: []APIServer{{
			URL:         listenURL,
			Description: "Local development server",
		}},
	}
}
