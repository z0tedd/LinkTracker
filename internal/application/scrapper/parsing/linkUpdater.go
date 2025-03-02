package parsing

import (
	"fmt"
)

// Interface LinkUpdater.
type linkUpdater interface {
	Update(link string) error
	GetData() map[string]string
}

// Factory function to create an instance of LinkUpdater.
func newLinkUpdater(host string) (linkUpdater, error) {
	switch host {
	case "github.com":
		return &gitHubLink{}, nil
	case "stackoverflow.com":
		return &stackOverflowLink{}, nil
	default:
		return nil, fmt.Errorf("unsupported host: %s", host)
	}
}

// Base structure for storing common data.
type baseLinkData struct {
	linkHost string
}
