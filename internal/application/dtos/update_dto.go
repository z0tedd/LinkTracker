package dtos

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type UpdateDTO struct {
	Description string  `json:"description"`
	ID          int64   `json:"ID,omitempty"`
	TgChatIDs   []int64 `json:"tgChatIDs,omitempty"`
	URL         string  `json:"url"`
}

func NewUpdateDTOFromSubscription(sub *domain.Subscription) (UpdateDTO, error) {
	answerPreview, err := pkg.StripHTMLTags(sub.LastActivity.AnswerPreview)
	if err != nil {
		return UpdateDTO{}, fmt.Errorf("creating dto: %w", err)
	}

	answerPreview = pkg.TruncateString(answerPreview, pkg.MaxPreviewLen)

	description := fmt.Sprintf(`
    Тема: %s
    Пользователь: %s
    Время: %s
    Превью комментария: %s 
  `, sub.LastActivity.Title, sub.LastActivity.Username,
		time.Unix(sub.LastActivity.DateUnix, 0).String(), answerPreview)

	return UpdateDTO{
		Description: description,
		ID:          sub.ID,
		TgChatIDs:   sub.TgChatIDs,
		URL:         sub.URL,
	}, nil
}

func (d *UpdateDTO) Decode(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *UpdateDTO) Encode() ([]byte, error) {
	return json.Marshal(d)
}
