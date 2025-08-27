package fetch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kb/internal"
)

// FetchArtifact reads a target and returns its contents as an Artifact. 
// Supports both local files and GitHub URLs (both public and private with token).
// For local files it reads the file directly. For GitHub URLs it uses the
// GitHub API to fetch content. Files with code extensions are wrapped in 
// fenced code blocks.
func FetchArtifact(target, githubToken string) (internal.Artifact, error) {
	// Handle GitHub URLs
	if IsGitHubURL(target) {
		return FetchGitHubArtifact(target, githubToken)
	}
	
	// Handle regular HTTP URLs (not GitHub)
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return internal.Artifact{}, errors.New("only GitHub URLs are supported for remote fetching")
	}
	
	// Handle local files
	return FetchLocalArtifact(target)
}

// FetchLocalArtifact handles local file fetching
func FetchLocalArtifact(target string) (internal.Artifact, error) {
	// strip "file://" prefix if present
	path := strings.TrimPrefix(target, "file://")
	b, err := os.ReadFile(path)
	if err != nil {
		return internal.Artifact{}, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	name := filepath.Base(path)
	text := string(b)
	md := FormatFileContent(text, filepath.Ext(path))
	
	return internal.Artifact{
		Source:       "local",
		CanonicalURL: "file://" + filepath.Clean(path),
		Title:        name,
		UpdatedAt:    time.Now(),
		Markdown:     md,
		Breadcrumbs:  nil,
	}, nil
}

// FormatFileContent wraps code files in fenced blocks based on file extension
func FormatFileContent(text, ext string) string {
	ext = strings.ToLower(ext)
	codeExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true, ".java": true, 
		".rb": true, ".rs": true, ".c": true, ".cpp": true, ".cs": true, 
		".sh": true, ".sql": true, ".yaml": true, ".yml": true, ".json": true, 
		".tf": true, ".html": true, ".css": true, ".scss": true, ".php": true,
		".r": true, ".scala": true, ".kt": true, ".swift": true,
	}
	
	if ext != "" && codeExts[ext] {
		// remove any trailing newlines to avoid double newlines around fences
		return fmt.Sprintf("```%s\n%s\n```\n", strings.TrimPrefix(ext, "."), strings.TrimRight(text, "\n"))
	}
	return text
}