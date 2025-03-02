package checker

import (
	"context"
	"fmt"
	"log"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/parsing"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/processing"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/repository"
)

func doSomething() {
	fmt.Println("I have done something!")
}

func checkSubscriptions(ctx context.Context, subs map[*domain.Subscription][]int64) ([]*domain.Subscription, error) {
	var updatedSubscriptions []*domain.Subscription

	githubClient, err := githubAPI.NewClientWithResponses("https://api.github.com")
	if err != nil {
		log.Print(err.Error())
		return nil, err
	}

	stackOverflowClient, err := stackOverflowAPI.NewClientWithResponses("https://api.stackexchange.com/2.3")
	if err != nil {
		log.Print(err.Error())
		return nil, err
	}

	for v := range subs {
		parsedLink, err := parsing.Link(v.Link)
		if err != nil {
			log.Print(err.Error())
			continue
		}

		switch parsedLink["linkHost"] {
		case "github":
			updatedSubscriptions = processing.Github(ctx, githubClient, parsedLink, v, updatedSubscriptions)
		case "stackoverflow":
			updatedSubscriptions = processing.StackOverflow(ctx, stackOverflowClient, parsedLink, v, updatedSubscriptions)
		}
	}

	return updatedSubscriptions, nil
}

func CheckLinks(repo repository.Repository, botClient botAPI.ClientWithResponsesInterface) {
	log.Println("I am checking!")

	ctx := context.Background()
	subscriptionsByUserIDs := repo.GetSubscriptionsByUserIDs()

	updatedSubscriptions, err := checkSubscriptions(ctx, subscriptionsByUserIDs)
	if err != nil {
		log.Println(err)
		return
	}

	for _, v := range updatedSubscriptions {
		description := fmt.Sprintf("Updated Link, url: %v ", v.Link)
		tgChatIDs := subscriptionsByUserIDs[v]
		URL := v.Link
		body := botAPI.PostUpdatesJSONRequestBody{
			Description: &description,
			Id:          nil,
			TgChatIds:   &tgChatIDs,
			Url:         &URL,
		}

		rsp, err := botClient.PostUpdatesWithResponse(ctx, body)
		if err != nil {
			log.Println(err, URL)
			continue
		}

		if rsp.JSON400 != nil {
			log.Println("Post response ended with error", rsp.JSON400.Description, rsp.JSON400)
			continue
		}
	}
}
