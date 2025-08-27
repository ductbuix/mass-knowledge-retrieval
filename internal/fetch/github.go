package fetch

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kb/internal"
)

// GitHubContent represents the response from GitHub's Contents API
type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	Content     string `json:"content,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

// GitHubClient handles authentication and API requests to GitHub
type GitHubClient struct {
	token      string
	httpClient *http.Client
}

// NewGitHubClient creates a new GitHub client with optional authentication
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// makeAPIRequest makes an authenticated request to the GitHub API
func (gh *GitHubClient) makeAPIRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	if gh.token != "" {
		req.Header.Set("Authorization", "token "+gh.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "kb-knowledge-retrieval/1.0")
	
	return gh.httpClient.Do(req)
}

// GetFileContent fetches a single file from a GitHub repository
func (gh *GitHubClient) GetFileContent(owner, repo, path, ref string) (*GitHubContent, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	if ref != "" {
		apiURL += "?ref=" + ref
	}
	
	resp, err := gh.makeAPIRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var content GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &content, nil
}

// GetDirectoryContents fetches all files in a directory from a GitHub repository
func (gh *GitHubClient) GetDirectoryContents(owner, repo, path, ref string) ([]GitHubContent, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	if ref != "" {
		apiURL += "?ref=" + ref
	}
	
	resp, err := gh.makeAPIRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return contents, nil
}

// GitHubURLInfo contains parsed GitHub URL components
type GitHubURLInfo struct {
	Owner string
	Repo  string
	Path  string
	Ref   string
}

// parseGitHubURL extracts components from various GitHub URL formats
func parseGitHubURL(gitURL string) (*GitHubURLInfo, error) {
	u, err := url.Parse(gitURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	
	if u.Host != "github.com" {
		return nil, errors.New("not a GitHub URL")
	}
	
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, errors.New("invalid GitHub URL format")
	}
	
	info := &GitHubURLInfo{
		Owner: parts[0],
		Repo:  parts[1],
	}
	
	// Handle different URL patterns
	if len(parts) >= 4 && parts[2] == "blob" {
		// https://github.com/owner/repo/blob/branch/path/to/file
		info.Ref = parts[3]
		if len(parts) > 4 {
			info.Path = strings.Join(parts[4:], "/")
		}
	} else if len(parts) >= 4 && parts[2] == "tree" {
		// https://github.com/owner/repo/tree/branch/path/to/dir
		info.Ref = parts[3]
		if len(parts) > 4 {
			info.Path = strings.Join(parts[4:], "/")
		}
	} else if len(parts) > 2 {
		// https://github.com/owner/repo/path/to/file (assume main branch)
		info.Ref = "main"
		info.Path = strings.Join(parts[2:], "/")
	}
	
	return info, nil
}

// IsGitHubURL checks if a URL is a GitHub URL
func IsGitHubURL(target string) bool {
	return strings.Contains(target, "github.com")
}

// FetchGitHubArtifact handles GitHub URL fetching
func FetchGitHubArtifact(target, token string) (internal.Artifact, error) {
	info, err := parseGitHubURL(target)
	if err != nil {
		return internal.Artifact{}, fmt.Errorf("invalid GitHub URL: %w", err)
	}
	
	client := NewGitHubClient(token)
	
	// If no path specified, fetch the entire repository root
	if info.Path == "" {
		return fetchGitHubDirectory(client, info, target)
	}
	
	// Try to fetch as a single file first
	content, err := client.GetFileContent(info.Owner, info.Repo, info.Path, info.Ref)
	if err != nil {
		// If single file fails, try as directory
		return fetchGitHubDirectory(client, info, target)
	}
	
	if content.Type != "file" {
		return fetchGitHubDirectory(client, info, target)
	}
	
	// Decode base64 content
	var text string
	if content.Encoding == "base64" && content.Content != "" {
		decoded, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			return internal.Artifact{}, fmt.Errorf("failed to decode base64 content: %w", err)
		}
		text = string(decoded)
	} else {
		text = content.Content
	}
	
	md := FormatFileContent(text, filepath.Ext(content.Name))
	
	return internal.Artifact{
		Source:       "github",
		CanonicalURL: target,
		Title:        content.Name,
		UpdatedAt:    time.Now(),
		Markdown:     md,
		Breadcrumbs:  []string{info.Owner, info.Repo, content.Path},
	}, nil
}

// fetchGitHubDirectory handles fetching multiple files from a GitHub directory
func fetchGitHubDirectory(client *GitHubClient, info *GitHubURLInfo, originalURL string) (internal.Artifact, error) {
	contents, err := client.GetDirectoryContents(info.Owner, info.Repo, info.Path, info.Ref)
	if err != nil {
		return internal.Artifact{}, fmt.Errorf("failed to fetch directory contents: %w", err)
	}
	
	var allContent strings.Builder
	var fileCount int
	
	for _, item := range contents {
		if item.Type == "file" {
			fileContent, err := client.GetFileContent(info.Owner, info.Repo, item.Path, info.Ref)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to fetch %s: %v\n", item.Path, err)
				continue
			}
			
			var text string
			if fileContent.Encoding == "base64" && fileContent.Content != "" {
				decoded, err := base64.StdEncoding.DecodeString(fileContent.Content)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to decode %s: %v\n", item.Path, err)
					continue
				}
				text = string(decoded)
			} else {
				text = fileContent.Content
			}
			
			if fileCount > 0 {
				allContent.WriteString("\n\n---\n\n")
			}
			
			allContent.WriteString(fmt.Sprintf("# %s\n\n", item.Path))
			allContent.WriteString(FormatFileContent(text, filepath.Ext(item.Name)))
			fileCount++
		}
	}
	
	if fileCount == 0 {
		return internal.Artifact{}, errors.New("no files found in directory")
	}
	
	title := fmt.Sprintf("%s/%s", info.Owner, info.Repo)
	if info.Path != "" {
		title += "/" + info.Path
	}
	
	return internal.Artifact{
		Source:       "github",
		CanonicalURL: originalURL,
		Title:        title,
		UpdatedAt:    time.Now(),
		Markdown:     allContent.String(),
		Breadcrumbs:  []string{info.Owner, info.Repo, info.Path},
	}, nil
}