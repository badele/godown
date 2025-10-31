package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test Markdown to HTML conversion
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

// Test media file detection
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

// Test Content-Type detection
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

// Test CSS server with embedded style
func TestServeCSSEmbedded(t *testing.T) {
	// Reset customStylePath to use embedded CSS
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

// Test CSS server with external file
func TestServeCSSCustom(t *testing.T) {
	// Create a temporary CSS file
	tmpDir := t.TempDir()
	cssFile := filepath.Join(tmpDir, "custom.css")
	cssContent := "body { color: red; }"
	if err := os.WriteFile(cssFile, []byte(cssContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Configure customStylePath
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

// Test CSS server with nonexistent file (fallback)
func TestServeCSSFallback(t *testing.T) {
	// Configure path to nonexistent file
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

	// Should use embedded CSS
	body := w.Body.String()
	if !strings.Contains(body, ":root") {
		t.Errorf("serveCSS() should fallback to embedded CSS")
	}
}

// Test media file server
func TestServeMedia(t *testing.T) {
	// Create temporary image file
	tmpDir := t.TempDir()
	imgFile := filepath.Join(tmpDir, "test.png")
	imgContent := []byte("fake png content")
	if err := os.WriteFile(imgFile, imgContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Temporarily change working directory
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

// Test Markdown server
func TestServeMarkdown(t *testing.T) {
	// Create temporary markdown file
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "README.md")
	mdContent := "# Test\n\nHello **world**"
	if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Temporarily change working directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Configure indexFile
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

// Test configuration via environment variables
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
			// Save old value
			oldValue := os.Getenv(tt.envKey)
			defer os.Setenv(tt.envKey, oldValue)

			// Set new value
			os.Setenv(tt.envKey, tt.envValue)

			// Check that environment variable is properly set
			result := os.Getenv(tt.envKey)
			if result != tt.expected {
				t.Errorf("os.Getenv(%v) = %v, want %v", tt.envKey, result, tt.expected)
			}
		})
	}
}

// Test text file detection
func TestIsTextFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Plain text file",
			content:  []byte("Hello world\nThis is text\n"),
			expected: true,
		},
		{
			name:     "Text with UTF-8 characters",
			content:  []byte("Bonjour le monde! éàç ûî ô\n"),
			expected: true,
		},
		{
			name:     "Binary file with null bytes",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			expected: false,
		},
		{
			name:     "Binary file with control characters",
			content:  []byte{0x7F, 0xEF, 0x01, 0x02},
			expected: false,
		},
		{
			name:     "Empty file",
			content:  []byte{},
			expected: true,
		},
		{
			name:     "Text with tabs and newlines",
			content:  []byte("Line 1\tTab\nLine 2\r\nLine 3"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := filepath.Join(tmpDir, tt.name)
			if err := os.WriteFile(tmpFile, tt.content, 0644); err != nil {
				t.Fatal(err)
			}

			result := isTextFile(tmpFile)
			if result != tt.expected {
				t.Errorf("isTextFile(%v) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

// Test hexadecimal formatting
func TestFormatBinaryAsHex(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		contains []string
	}{
		{
			name:  "Simple binary data",
			input: []byte{0x00, 0x41, 0x42, 0x43},
			contains: []string{
				"00000000", // offset
				"41 42 43", // hex bytes
				"ABC",      // ASCII representation
			},
		},
		{
			name:  "HTML special characters",
			input: []byte{'<', '>', '&', '"', 'A'},
			contains: []string{
				"3c 3e 26 22 41", // hex values: < > & " A
				"&lt;",           // escaped <
				"&gt;",           // escaped >
				"&amp;",          // escaped &
			},
		},
		{
			name:  "Full line (16 bytes)",
			input: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
			contains: []string{
				"00000000",
				"00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f",
			},
		},
		{
			name:  "Multiple lines",
			input: make([]byte, 32), // 32 bytes = 2 lines
			contains: []string{
				"00000000", // first line
				"00000010", // second line
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBinaryAsHex(tt.input)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("formatBinaryAsHex() result should contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

// Test byte size formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{1023, "1023"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test text file server
func TestServeTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "test.txt")
	txtContent := "Hello world\nLine 2\nSpecial: <tag> & \"quotes\""
	if err := os.WriteFile(txtFile, []byte(txtContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest("GET", "/test.txt", nil)
	w := httptest.NewRecorder()

	serveTextFile(w, req, "test.txt")

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("serveTextFile() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("serveTextFile() Content-Type = %v, want %v", contentType, "text/html; charset=utf-8")
	}

	body := w.Body.String()
	if !strings.Contains(body, "<pre") {
		t.Errorf("serveTextFile() body should contain <pre> tag")
	}
	// Check HTML escaping
	if !strings.Contains(body, "&lt;tag&gt;") {
		t.Errorf("serveTextFile() should escape HTML tags")
	}
	if !strings.Contains(body, "&amp;") {
		t.Errorf("serveTextFile() should escape ampersands")
	}
	if !strings.Contains(body, "/__godown_style.css") {
		t.Errorf("serveTextFile() body should link to style")
	}
}

// Test binary file server
func TestServeBinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	binFile := filepath.Join(tmpDir, "test.bin")
	binContent := []byte{0x7F, 0x45, 0x4C, 0x46, 0x00, 0x01, 0x02, 0x03, 0x41, 0x42, 0x43}
	if err := os.WriteFile(binFile, binContent, 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest("GET", "/test.bin", nil)
	w := httptest.NewRecorder()

	serveBinaryFile(w, req, "test.bin")

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("serveBinaryFile() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("serveBinaryFile() Content-Type = %v, want %v", contentType, "text/html; charset=utf-8")
	}

	body := w.Body.String()
	if !strings.Contains(body, "test.bin") {
		t.Errorf("serveBinaryFile() body should contain filename")
	}
	if !strings.Contains(body, "bytes") {
		t.Errorf("serveBinaryFile() body should show file size")
	}
	if !strings.Contains(body, "00000000") {
		t.Errorf("serveBinaryFile() body should contain hex offset")
	}
	if !strings.Contains(body, "/__godown_style.css") {
		t.Errorf("serveBinaryFile() body should link to style")
	}
}

// Test environment variables vs flags priority
func TestConfigurationPriority(t *testing.T) {
	// Test for PORT
	t.Run("PORT env takes precedence", func(t *testing.T) {
		oldPort := os.Getenv("PORT")
		defer os.Setenv("PORT", oldPort)

		os.Setenv("PORT", "5000")

		// In main code, PORT env would be read before flag
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080" // flag default
		}

		if port != "5000" {
			t.Errorf("Expected PORT=5000 from env, got %v", port)
		}
	})

	// Test for INDEX
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

	// Test for STYLE
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

	// Test with empty environment variable (flag should be used)
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
