package parsing

import (
	"fmt"
	"net/url"
	"strings"
)

// Structure for GitHub.
type gitHubLink struct {
	baseLinkData
	Owner string
	Repo  string
}

func (g *gitHubLink) Update(link string) error {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	path := strings.Trim(parsedURL.Path, "/")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid GitHub path: %s", path)
	}

	g.linkHost = "github"
	g.Owner = parts[0]
	g.Repo = parts[1]

	return nil
}

func (g *gitHubLink) GetData() map[string]string {
	return map[string]string{
		"linkHost": g.linkHost,
		"owner":    g.Owner,
		"repo":     g.Repo,
	}
}
