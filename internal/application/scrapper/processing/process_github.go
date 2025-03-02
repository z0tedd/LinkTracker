package processing

import (
	"context"
	"fmt"
	"net/http"
	"time"

	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

func Github(ctx context.Context, githubClient githubAPI.ClientWithResponsesInterface,
	parsedLink map[string]string, currentSub *domain.Subscription, updatedSubscriptions []*domain.Subscription,
) []*domain.Subscription {
	// Make the API call
	rsp, err := githubClient.GetReposOwnerRepoWithResponse(ctx, parsedLink["owner"], parsedLink["repo"])
	if err != nil {
		// Log the error and return early
		fmt.Printf("Error making request: %v\n", err)
		return updatedSubscriptions
	}

	// Check if the response is nil
	if rsp == nil || rsp.HTTPResponse == nil {
		fmt.Println("Received nil response")
		return updatedSubscriptions
	}

	// Log the status code
	fmt.Printf("Status Code: %d\n", rsp.StatusCode())
	// Handle non-200 status codes
	if rsp.StatusCode() != http.StatusOK {
		fmt.Printf("Unexpected status code: %d\n", rsp.StatusCode())
		fmt.Printf("Response body: %s\n", string(rsp.Body))

		return updatedSubscriptions
	}

	// Process the response body
	if rsp.JSON200 != nil {
		repoInfo := rsp.JSON200
		lastActivityTime := time.Unix(currentSub.LastActivityDate, 0)

		// Check if the repository has been updated
		if repoInfo.UpdatedAt.After(lastActivityTime) {
			updatedSubscriptions = append(updatedSubscriptions, currentSub)
			currentSub.LastActivityDate = repoInfo.UpdatedAt.Unix()
		} else {
			fmt.Println("No answers found.")
		}
	} else {
		fmt.Println("Empty JSON200 response")
	}

	return updatedSubscriptions
}
