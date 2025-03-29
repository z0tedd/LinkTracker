package pkg

import (
	"strings"

	"golang.org/x/net/html"
)

// Function to strip HTML tags and return plain text.
func StripHTMLTags(htmlContent string) (string, error) {
	var sb strings.Builder

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	// Recursive function to traverse the HTML nodes
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	extractText(doc)

	return sb.String(), nil
}
