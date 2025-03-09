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
)

// subs map[*domain.Subscription][]int64.
type Repository interface {
	GetSubsID() domain.Set
	GetSubscription(subID int64) (domain.Subscription, error)
	UpdateSubscriptionActivity(subID int64, newActivity domain.Activity) error
}
type Checker struct {
	githubClient        githubAPI.ClientWithResponsesInterface
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface
	botClient           botAPI.ClientWithResponsesInterface
}

func NewChecker(
	githubClient githubAPI.ClientWithResponsesInterface,
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface,
	botClient botAPI.ClientWithResponsesInterface,
) *Checker {
	return &Checker{githubClient, stackOverflowClient, botClient}
}

func (c Checker) CheckSubscriptions(ctx context.Context, repo Repository) {
	log.Print("i am checking!\n")

	for subID := range repo.GetSubsID() {
		sub, err := repo.GetSubscription(subID)
		if err != nil {
			log.Print(err.Error())
			continue
		}

		parsedLink, err := parsing.Link(sub.URL)
		if err != nil {
			log.Print(err.Error())
			continue
		}

		switch parsedLink["linkHost"] {
		case "github":
			if newActivity, updated := processing.Github(ctx, c.githubClient, parsedLink, subID, repo); updated {
				err = repo.UpdateSubscriptionActivity(subID, newActivity)
				if err != nil {
					log.Print(err.Error())
					continue
				}
			}
		case "stackoverflow":
			if newActivity, updated := processing.StackOverflow(ctx, c.stackOverflowClient, parsedLink, subID, repo); updated {
				err = repo.UpdateSubscriptionActivity(subID, newActivity)
				if err != nil {
					log.Print(err.Error())
					continue
				}
			}
		}

		sub, err = repo.GetSubscription(subID)
		if err != nil {
			log.Print(err.Error())
			continue
		}

		err = processUpdatedSubscription(ctx, c.botClient, sub)
		if err != nil {
			log.Print(err.Error())
		}
	}
}

func processUpdatedSubscription(ctx context.Context, botClient botAPI.ClientWithResponsesInterface,
	subscription domain.Subscription,
) error {
	body := botAPI.PostUpdatesJSONRequestBody{
		Description: &subscription.UpdateDescription,
		Id:          &subscription.ID,
		TgChatIds:   &subscription.TgChatIDs,
		Url:         &subscription.URL,
	}

	rsp, err := botClient.PostUpdatesWithResponse(ctx, body)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	if rsp.JSON400 != nil && rsp.JSON400.Description != nil {
		return domain.StatusCode400Error{Msg: fmt.Sprintf("code: 400, description: %s", *rsp.JSON400.Description)}
	}

	return nil
}

// // CheckLinks Берет
// func CheckLinks(repo repository.Repository, botClient botAPI.ClientWithResponsesInterface) {
// 	log.Println("I am checking!")
//
// 	ctx := context.Background()
// 	subscriptionsByUserIDs := repo.GetSubscriptionsByUserIDs()
//
// 	updatedSubscriptions, err := checkSubs(ctx, subscriptionsByUserIDs)
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}
//
// 	for _, subscription := range updatedSubscriptions {
// 		if err = processUpdatedSubscription(ctx, botClient, subscription); err != nil {
// 			log.Println(err, subscription.URL)
// 		}
// 	}
// }
