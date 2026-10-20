package httpapi

import (
	_ "embed"
	"net/http"
)

// The spec is compiled into the binary, so the docs work wherever the API runs.
//
//go:embed openapi.yaml
var openapiYAML []byte

const swaggerPage = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Warehouse Stock Ledger API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>SwaggerUIBundle({ url: "/api/openapi.yaml", dom_id: "#swagger-ui" });</script>
</body>
</html>`

func (s *Server) openapiSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openapiYAML)
}

func (s *Server) docsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerPage))
}
