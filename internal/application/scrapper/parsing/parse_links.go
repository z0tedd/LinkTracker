package parsing

import (
	"fmt"
	"net/url"
)

// Main ParseLink function.
func Link(link string) (map[string]string, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Hostname()

	updater, err := newLinkUpdater(host)
	if err != nil {
		return nil, err
	}

	if err := updater.Update(link); err != nil {
		return nil, err
	}

	return updater.GetData(), nil
}
