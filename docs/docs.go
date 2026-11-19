package docs

import (
	"strings"

	"github.com/swaggo/swag"
)

const docTemplate = `{
  "swagger": "2.0",
  "info": {
    "title": "Delcom Blog API",
    "description": "Dokumentasi REST API Delcom Blog Multi-Tier & AI Generator",
    "version": "1.0"
  },
  "host": "{{.Host}}",
  "basePath": "{{.BasePath}}",
  "schemes": [
    "http",
    "https"
  ],
  "paths": {
    "/public/categories": {
      "get": {
        "summary": "Daftar Kategori Publik",
        "responses": {
          "200": { "description": "Success" }
        }
      }
    },
    "/public/posts": {
      "get": {
        "summary": "Daftar Artikel Publik",
        "parameters": [
          { "name": "category", "in": "query", "type": "string" },
          { "name": "q", "in": "query", "type": "string" }
        ],
        "responses": {
          "200": { "description": "Success" }
        }
      }
    },
    "/auth/register": {
      "post": {
        "summary": "Registrasi Pengguna",
        "responses": {
          "201": { "description": "Created" }
        }
      }
    },
    "/auth/login": {
      "post": {
        "summary": "Login Pengguna",
        "responses": {
          "200": { "description": "Success" }
        }
      }
    },
    "/posts": {
      "get": {
        "summary": "Daftar Artikel Penulis",
        "responses": {
          "200": { "description": "Success" }
        }
      },
      "post": {
        "summary": "Tambah Artikel",
        "responses": {
          "201": { "description": "Created" }
        }
      }
    },
    "/posts/ai-generate": {
      "post": {
        "summary": "Generate Draf Artikel AI",
        "responses": {
          "200": { "description": "Success" }
        }
      }
    }
  }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8081",
	BasePath:         "/api",
	Schemes:          []string{"http", "https"},
	Title:            "Delcom Blog API",
	Description:      "Dokumentasi REST API Delcom Blog Multi-Tier & AI Generator",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}

func GetSwaggerJSON(host, scheme string) (string, error) {
	if host != "" {
		SwaggerInfo.Host = host
	}
	if scheme != "" {
		scheme = strings.ToLower(scheme)
		SwaggerInfo.Schemes = []string{scheme}
	}
	doc := SwaggerInfo.ReadDoc()
	return doc, nil
}
