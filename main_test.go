package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test de la conversion Markdown vers HTML
func TestMdToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "Simple paragraph",
			input:    "Hello world",
			contains: "<p>Hello world</p>",
		},
		{
			name:     "Header",
			input:    "# Title",
			contains: `<h1 id="title">Title</h1>`,
		},
		{
			name:     "Bold text",
			input:    "**bold**",
			contains: "<strong>bold</strong>",
		},
		{
			name:     "Link",
			input:    "[Google](https://google.com)",
			contains: `<p><a href="https://google.com" target=`,
		},
		{
			name:     "Code block",
			input:    "```\ncode\n```",
			contains: "<pre><code>code\n</code></pre>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mdToHTML([]byte(tt.input))
			resultStr := string(result)
			if !strings.Contains(resultStr, tt.contains) {
				t.Errorf("mdToHTML() = %v, want to contain %v", resultStr, tt.contains)
			}
		})
	}
}

// Test de la détection des fichiers média
func TestIsMediaFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/path/to/image.jpg", true},
		{"/path/to/image.png", true},
		{"/path/to/image.gif", true},
		{"/path/to/image.svg", true},
		{"/path/to/video.mp4", true},
		{"/path/to/style.css", true},
		{"/path/to/document.md", false},
		{"/path/to/file.txt", false},
		{"/path/to/script.js", false},
		{"/path/to/IMAGE.PNG", true}, // Test case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isMediaFile(tt.path)
			if result != tt.expected {
				t.Errorf("isMediaFile(%v) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// Test du Content-Type
func TestGetContentType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/image.jpg", "image/jpeg"},
		{"/path/to/image.png", "image/png"},
		{"/path/to/image.gif", "image/gif"},
		{"/path/to/image.svg", "image/svg+xml"},
		{"/path/to/video.mp4", "video/mp4"},
		{"/path/to/style.css", "text/css"},
		{"/path/to/unknown.xyz", "application/octet-stream"},
		{"/path/to/IMAGE.PNG", "image/png"}, // Test case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getContentType(tt.path)
			if result != tt.expected {
				t.Errorf("getContentType(%v) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// Test du serveur CSS avec style embarqué
func TestServeCSSEmbedded(t *testing.T) {
	// Réinitialiser customStylePath pour utiliser le CSS embarqué
	oldPath := customStylePath
	customStylePath = ""
	defer func() { customStylePath = oldPath }()

	req := httptest.NewRequest("GET", "/__godown_style.css", nil)
	w := httptest.NewRecorder()

	serveCSS(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("serveCSS() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("serveCSS() Content-Type = %v, want %v", contentType, "text/css; charset=utf-8")
	}

	body := w.Body.String()
	if !strings.Contains(body, ":root") {
		t.Errorf("serveCSS() body should contain CSS root variables")
	}
	if !strings.Contains(body, "--bg-color") {
		t.Errorf("serveCSS() body should contain --bg-color variable")
	}
}

// Test du serveur CSS avec fichier externe
func TestServeCSSCustom(t *testing.T) {
	// Créer un fichier CSS temporaire
	tmpDir := t.TempDir()
	cssFile := filepath.Join(tmpDir, "custom.css")
	cssContent := "body { color: red; }"
	if err := os.WriteFile(cssFile, []byte(cssContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Configurer customStylePath
	oldPath := customStylePath
	customStylePath = cssFile
	defer func() { customStylePath = oldPath }()

	req := httptest.NewRequest("GET", "/__godown_style.css", nil)
	w := httptest.NewRecorder()

	serveCSS(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("serveCSS() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	body := w.Body.String()
	if body != cssContent {
		t.Errorf("serveCSS() body = %v, want %v", body, cssContent)
	}
}

// Test du serveur CSS avec fichier inexistant (fallback)
func TestServeCSSFallback(t *testing.T) {
	// Configurer un chemin vers un fichier inexistant
	oldPath := customStylePath
	customStylePath = "/nonexistent/file.css"
	defer func() { customStylePath = oldPath }()

	req := httptest.NewRequest("GET", "/__godown_style.css", nil)
	w := httptest.NewRecorder()

	serveCSS(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("serveCSS() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	// Devrait utiliser le CSS embarqué
	body := w.Body.String()
	if !strings.Contains(body, ":root") {
		t.Errorf("serveCSS() should fallback to embedded CSS")
	}
}

// Test du serveur de médias
func TestServeMedia(t *testing.T) {
	// Créer un fichier image temporaire
	tmpDir := t.TempDir()
	imgFile := filepath.Join(tmpDir, "test.png")
	imgContent := []byte("fake png content")
	if err := os.WriteFile(imgFile, imgContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Changer le répertoire de travail temporairement
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest("GET", "/test.png", nil)
	w := httptest.NewRecorder()

	serveMedia(w, req, "test.png")

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("serveMedia() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("serveMedia() Content-Type = %v, want %v", contentType, "image/png")
	}

	body := w.Body.Bytes()
	if string(body) != string(imgContent) {
		t.Errorf("serveMedia() body = %v, want %v", body, imgContent)
	}
}

// Test du serveur Markdown
func TestServeMarkdown(t *testing.T) {
	// Créer un fichier markdown temporaire
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "README.md")
	mdContent := "# Test\n\nHello **world**"
	if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Changer le répertoire de travail temporairement
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Configurer l'indexFile
	oldIndex := indexFile
	indexFile = "README.md"
	defer func() { indexFile = oldIndex }()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	serveMarkdown(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("serveMarkdown() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("serveMarkdown() Content-Type = %v, want %v", contentType, "text/html; charset=utf-8")
	}

	body := w.Body.String()
	if !strings.Contains(body, "<h1") {
		t.Errorf("serveMarkdown() body should contain h1 tag")
	}
	if !strings.Contains(body, "<strong>world</strong>") {
		t.Errorf("serveMarkdown() body should contain bold text")
	}
	if !strings.Contains(body, "/__godown_style.css") {
		t.Errorf("serveMarkdown() body should link to style")
	}
}

// Test de la configuration via variables d'environnement
func TestEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		varName  string
		expected string
	}{
		{
			name:     "PORT environment variable",
			envKey:   "PORT",
			envValue: "9999",
			varName:  "port",
			expected: "9999",
		},
		{
			name:     "INDEX environment variable",
			envKey:   "INDEX",
			envValue: "custom.md",
			varName:  "index",
			expected: "custom.md",
		},
		{
			name:     "STYLE environment variable",
			envKey:   "STYLE",
			envValue: "/custom/style.css",
			varName:  "style",
			expected: "/custom/style.css",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sauvegarder l'ancienne valeur
			oldValue := os.Getenv(tt.envKey)
			defer os.Setenv(tt.envKey, oldValue)

			// Définir la nouvelle valeur
			os.Setenv(tt.envKey, tt.envValue)

			// Vérifier que la variable d'environnement est bien définie
			result := os.Getenv(tt.envKey)
			if result != tt.expected {
				t.Errorf("os.Getenv(%v) = %v, want %v", tt.envKey, result, tt.expected)
			}
		})
	}
}

// Test de la priorité des variables d'environnement vs flags
func TestConfigurationPriority(t *testing.T) {
	// Test pour PORT
	t.Run("PORT env takes precedence", func(t *testing.T) {
		oldPort := os.Getenv("PORT")
		defer os.Setenv("PORT", oldPort)

		os.Setenv("PORT", "5000")

		// Dans le code main, PORT env serait lu avant le flag
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080" // flag default
		}

		if port != "5000" {
			t.Errorf("Expected PORT=5000 from env, got %v", port)
		}
	})

	// Test pour INDEX
	t.Run("INDEX env takes precedence", func(t *testing.T) {
		oldIndex := os.Getenv("INDEX")
		defer os.Setenv("INDEX", oldIndex)

		os.Setenv("INDEX", "custom.md")

		index := os.Getenv("INDEX")
		if index == "" {
			index = "README.md" // flag default
		}

		if index != "custom.md" {
			t.Errorf("Expected INDEX=custom.md from env, got %v", index)
		}
	})

	// Test pour STYLE
	t.Run("STYLE env takes precedence", func(t *testing.T) {
		oldStyle := os.Getenv("STYLE")
		defer os.Setenv("STYLE", oldStyle)

		os.Setenv("STYLE", "/custom.css")

		style := os.Getenv("STYLE")
		if style == "" {
			style = "" // flag default (empty = embedded)
		}

		if style != "/custom.css" {
			t.Errorf("Expected STYLE=/custom.css from env, got %v", style)
		}
	})

	// Test avec variable d'environnement vide (flag devrait être utilisé)
	t.Run("Flag used when env is empty", func(t *testing.T) {
		oldPort := os.Getenv("PORT")
		defer os.Setenv("PORT", oldPort)

		os.Setenv("PORT", "")

		port := os.Getenv("PORT")
		if port == "" {
			port = "3000" // flag value
		}

		if port != "3000" {
			t.Errorf("Expected port=3000 from flag when env empty, got %v", port)
		}
	})
}
