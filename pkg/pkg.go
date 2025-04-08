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

func TruncateString(input string, limit int) string {
	// Проверяем, не превышает ли длина строки лимит
	if len(input) > limit {
		// Обрезаем строку до указанного лимита
		return input[:limit]
	}
	// Если строка короче или равна лимиту, возвращаем её без изменений
	return input
}
func StringPtr(s string) *string { return &s }

// Helper function to convert a map[string]string into an array of strings.
func ConvertToArray(m map[string]string) []string {
	result := make([]string, 0, len(m)) // Preallocate space for efficiency

	for key, value := range m {
		// Format the key-value pair as "key: value" and append to the result array
		result = append(result, key+": "+value)
	}

	return result
}

// Helper function to convert an array of strings into a map.
func ConvertToMap(arr []string) map[string]string {
	result := make(map[string]string)

	for _, item := range arr {
		// Split the string by the colon (":") delimiter
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])   // Trim any extra spaces around the key
			value := strings.TrimSpace(parts[1]) // Trim any extra spaces around the value
			result[key] = value
		}
	}

	return result
}
