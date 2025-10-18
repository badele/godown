package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="{{.StylePath}}">
</head>
<body>
    {{.Content}}
</body>
</html>`

const defaultCSS = `/* Variables pour le thème light */
:root {
    --bg-color: #ffffff;
    --text-color: #333333;
    --link-color: #0366d6;
    --border-color: #dddddd;
    --code-bg: #f5f5f5;
    --quote-border: #dddddd;
    --quote-text: #666666;
    --table-header-bg: #f5f5f5;
}

/* Variables pour le thème dark */
@media (prefers-color-scheme: dark) {
    :root {
        --bg-color: #1e1e1e;
        --text-color: #e0e0e0;
        --link-color: #58a6ff;
        --border-color: #444444;
        --code-bg: #2d2d2d;
        --quote-border: #444444;
        --quote-text: #aaaaaa;
        --table-header-bg: #2d2d2d;
    }
}

body {
    max-width: 900px;
    margin: 40px auto;
    padding: 0 20px;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    line-height: 1.6;
    color: var(--text-color);
    background-color: var(--bg-color);
    transition: background-color 0.3s ease, color 0.3s ease;
}

pre {
    background: var(--code-bg);
    padding: 15px;
    overflow-x: auto;
    border-radius: 5px;
    border: 1px solid var(--border-color);
}

code {
    background: var(--code-bg);
    padding: 2px 6px;
    border-radius: 3px;
    font-family: 'Courier New', monospace;
}

pre code {
    background: none;
    padding: 0;
}

a {
    color: var(--link-color);
    text-decoration: none;
}

a:hover {
    text-decoration: underline;
}

h1, h2, h3 {
    margin-top: 24px;
}

table {
    border-collapse: collapse;
    width: 100%;
}

th, td {
    border: 1px solid var(--border-color);
    padding: 8px;
    text-align: left;
}

th {
    background: var(--table-header-bg);
}

blockquote {
    border-left: 4px solid var(--quote-border);
    margin: 0;
    padding-left: 16px;
    color: var(--quote-text);
}

img {
    max-width: 100%;
    height: auto;
}
`

var (
	tmpl            = template.Must(template.New("page").Parse(htmlTemplate))
	customStylePath string
	indexFile       string
	defaultPort     = "8080"
)

type PageData struct {
	Title     string
	Content   template.HTML
	StylePath string
}

func mdToHTML(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.Tables | parser.FencedCode
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

// isMediaFile checks if the file is a media file (image, svg, video) or a static file
func isMediaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	mediaExtensions := []string{
		// Images
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".ico",
		// SVG
		".svg",
		// Videos
		".mp4", ".webm", ".ogg", ".avi", ".mov", ".mkv",
		// CSS
		".css",
	}

	for _, mediaExt := range mediaExtensions {
		if ext == mediaExt {
			return true
		}
	}
	return false
}

// getContentType returns the appropriate Content-Type for a file
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	contentTypes := map[string]string{
		// Images
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
		".ico":  "image/x-icon",
		".svg":  "image/svg+xml",
		// Videos
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".ogg":  "video/ogg",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".mkv":  "video/x-matroska",
		// CSS
		".css": "text/css",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}

// serveCSS serves the stylesheet (embedded or external)
func serveCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")

	if customStylePath == "" {
		// Use embedded CSS
		w.Write([]byte(defaultCSS))
	} else {
		// Use external CSS file
		content, err := os.ReadFile(customStylePath)
		if err != nil {
			log.Printf("Error reading custom CSS file %s: %v, falling back to embedded CSS", customStylePath, err)
			w.Write([]byte(defaultCSS))
			return
		}
		w.Write(content)
	}
}

// serveMedia serves a media file
func serveMedia(w http.ResponseWriter, r *http.Request, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// Set appropriate Content-Type
	w.Header().Set("Content-Type", getContentType(filePath))

	// Copy file to response
	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("Error serving media file %s: %v", filePath, err)
	}
}

func serveMarkdown(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/" + indexFile
	}

	// Clean the path
	path = filepath.Clean(path)

	// Check if the requested file is a media file
	filePath := filepath.Join(".", path)
	if isMediaFile(filePath) {
		serveMedia(w, r, filePath)
		return
	}

	// For markdown files
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
		filePath = filepath.Join(".", path)
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		// Try without .md for directories
		dirPath := strings.TrimSuffix(filePath, ".md")
		readmePath := filepath.Join(dirPath, "README.md")
		content, err = os.ReadFile(readmePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		filePath = readmePath
	}

	// Convert and render
	htmlContent := mdToHTML(content)
	title := filepath.Base(filePath)

	data := PageData{
		Title:     title,
		Content:   template.HTML(htmlContent),
		StylePath: "/__godown_style.css",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func main() {
	// Define flags
	portFlag := flag.String("port", defaultPort, "HTTP server port (or PORT env var)")
	styleFlag := flag.String("style", "", "Custom CSS file path (or STYLE env var)")
	indexFlag := flag.String("index", "README.md", "Default index file (or INDEX env var)")
	flag.Parse()

	// Priority: environment variable > flag > default
	port := os.Getenv("PORT")
	if port == "" {
		port = *portFlag
	}

	customStylePath = os.Getenv("STYLE")
	if customStylePath == "" {
		customStylePath = *styleFlag
	}

	indexFile = os.Getenv("INDEX")
	if indexFile == "" {
		indexFile = *indexFlag
	}

	// Display CSS mode
	if customStylePath == "" {
		log.Printf("Using embedded CSS")
	} else {
		log.Printf("Using custom CSS: %s", customStylePath)
	}

	// Routes
	http.HandleFunc("/__godown_style.css", serveCSS)
	http.HandleFunc("/", serveMarkdown)

	log.Printf("Serving Markdown files on http://localhost:%s", port)
	log.Printf("Index: %s", indexFile)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
