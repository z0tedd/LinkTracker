package dtos

import (
	"encoding/json"
)

type DeleteLinksDTO struct {
	TgChatID int64  `json:"TgChatID"`
	Link     string `json:"Link"`
}

func NewDeleteLinksDTO(chatID int64, link string) *DeleteLinksDTO {
	return &DeleteLinksDTO{TgChatID: chatID, Link: link}
}

func (d *DeleteLinksDTO) Decode(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *DeleteLinksDTO) Encode() ([]byte, error) {
	return json.Marshal(d)
}

type GetLinksDTO struct {
	TgChatID int64 `json:"TgChatID"`
}

func NewGetLinksDTO(chatID int64) *GetLinksDTO {
	return &GetLinksDTO{TgChatID: chatID}
}

func (d *GetLinksDTO) Decode(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *GetLinksDTO) Encode() ([]byte, error) {
	return json.Marshal(d)
}

type PostLinksDTO struct {
	TgChatID int64    `json:"tgChatID"`
	Filters  []string `json:"filters,omitempty"`
	Link     string   `json:"link"`
	Tags     []string `json:"tags,omitempty"`
}

func NewPostLinksDTO(chatID int64, filters []string, link string, tags []string) *PostLinksDTO {
	return &PostLinksDTO{TgChatID: chatID, Link: link, Tags: tags, Filters: filters}
}

func (d *PostLinksDTO) Decode(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *PostLinksDTO) Encode() ([]byte, error) {
	return json.Marshal(d)
}
