package processing

import (
	"context"
	"fmt"

	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

func StackOverflow(
	ctx context.Context,
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface,
	parsedLink map[string]string,
	currentSub *domain.Subscription,
	updatedSubscriptions []*domain.Subscription,
) []*domain.Subscription {
	// Параметры для запроса
	params := stackOverflowAPI.GetQuestionsByIdsParams{
		Site: "stackoverflow",
	}

	// Выполнение запроса к API
	rsp, err := stackOverflowClient.GetQuestionsByIdsWithResponse(ctx, parsedLink["questionID"], &params)
	if err != nil || rsp.StatusCode() != 200 {
		fmt.Println("error: ", err)
		return nil
	}

	// Проверка наличия данных в ответе
	if rsp.JSON200 == nil || rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		fmt.Println("Question not found or invalid response.")
		return updatedSubscriptions
	}

	// Получение информации о вопросе
	questionInfo := rsp.JSON200
	currentActivityTime := *(*questionInfo.Items)[0].LastActivityDate

	// Проверка времени последней активности
	if int64(currentActivityTime) > currentSub.LastActivityDate {
		updatedSubscriptions = append(updatedSubscriptions, currentSub)
		currentSub.LastActivityDate = int64(currentActivityTime)
	}

	return updatedSubscriptions
}
