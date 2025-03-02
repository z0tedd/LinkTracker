package parsing

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Structure for Stack Overflow.
type stackOverflowLink struct {
	baseLinkData
	QuestionID string
}

func (s *stackOverflowLink) Update(link string) error {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	path := strings.Trim(parsedURL.Path, "/")
	re := regexp.MustCompile(`^questions/(\d+)`)

	matches := re.FindStringSubmatch(path)
	if len(matches) < 2 {
		return fmt.Errorf("invalid Stack Overflow path: %s", path)
	}

	s.linkHost = "stackoverflow"
	s.QuestionID = matches[1]

	return nil
}

func (s *stackOverflowLink) GetData() map[string]string {
	return map[string]string{
		"linkHost":   s.linkHost,
		"questionID": s.QuestionID,
	}
}
