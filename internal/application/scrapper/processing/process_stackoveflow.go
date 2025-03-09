package processing

import (
	"context"
	"fmt"
	"log"

	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

func StackOverflow(
	ctx context.Context, stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface,
	parsedLink map[string]string, subID int64, repo RepositoryWithActivity,
) (domain.Activity, bool) {
	// Параметры для запроса
	params := stackOverflowAPI.GetQuestionsByIdsParams{
		Site: "stackoverflow",
	}

	sub, err := repo.GetSubscription(subID)
	if err != nil {
		log.Print("Process: ", err.Error())
		return domain.Activity{}, false
	}
	// Выполнение запроса к API
	rsp, err := stackOverflowClient.GetQuestionsByIdsWithResponse(ctx, parsedLink["questionID"], &params)
	if err != nil || rsp.StatusCode() != 200 {
		fmt.Println("error: ", err)
		return sub.LastActivity, false
	}

	// Проверка наличия данных в ответе
	if rsp.JSON200 == nil || rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		fmt.Println("Question not found or invalid response.")
		return sub.LastActivity, false
	}

	// Получение информации о вопросе
	questionInfo := rsp.JSON200
	currentActivityTime := *(*questionInfo.Items)[0].LastActivityDate

	// Проверка времени последней активности
	if int64(currentActivityTime) > sub.LastActivity.DateUnix {
		sub.LastActivity.DateUnix = int64(currentActivityTime)
		return sub.LastActivity, true
	}

	return sub.LastActivity, false
}
