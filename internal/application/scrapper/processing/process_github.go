package processing

import (
	"context"
	"fmt"
	"log"
	"net/http"

	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type RepositoryWithActivity interface {
	GetSubscription(subID int64) (domain.Subscription, error)
}

func Github(ctx context.Context, githubClient githubAPI.ClientWithResponsesInterface,
	parsedLink map[string]string, subID int64, repo RepositoryWithActivity,
) (domain.Activity, bool) {
	updated := false

	sub, err := repo.GetSubscription(subID)
	if err != nil {
		log.Print("Process: ", err.Error())
		return domain.Activity{}, updated
	}

	lastActivity := sub.LastActivity

	// Make the API call
	rsp, err := githubClient.GetReposOwnerRepoWithResponse(ctx, parsedLink["owner"], parsedLink["repo"])
	if err != nil {
		// Log the error and return early
		fmt.Printf("Error making request: %v\n", err)
		return lastActivity, updated
	}

	// Check if the response is nil
	if rsp == nil || rsp.HTTPResponse == nil {
		fmt.Println("Received nil response")
		return lastActivity, updated
	}

	// Log the status code
	fmt.Printf("Status Code: %d\n", rsp.StatusCode())
	// Handle non-200 status codes
	if rsp.StatusCode() != http.StatusOK {
		fmt.Printf("Unexpected status code: %d\n", rsp.StatusCode())
		fmt.Printf("Response body: %s\n", string(rsp.Body))

		return lastActivity, updated
	}

	// Process the response body
	if rsp.JSON200 != nil {
		repoInfo := rsp.JSON200
		// Check if the repository has been updated
		if repoInfo.UpdatedAt.Unix() > lastActivity.DateUnix {
			updated = true
			lastActivity.DateUnix = repoInfo.UpdatedAt.Unix()
		} else {
			fmt.Println("No answers found.")
		}
	} else {
		fmt.Println("Empty JSON200 response")
	}

	return lastActivity, updated
}
