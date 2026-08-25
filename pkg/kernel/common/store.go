package common

import "encoding/json"

type StoreQueryRequest struct {
	RequestId         string         `json:"requestId"`
	IncludeData       bool           `json:"includeData"`
	PubsubTopic       string         `json:"pubsubTopic,omitempty"`
	ContentTopics     *[]string      `json:"contentTopics,omitempty"`
	TimeStart         *int64         `json:"timeStart,omitempty"`
	TimeEnd           *int64         `json:"timeEnd,omitempty"`
	MessageHashes     *[]MessageHash `json:"messageHashes,omitempty"`
	PaginationCursor  *MessageHash   `json:"paginationCursor,omitempty"`
	PaginationForward bool           `json:"paginationForward"`
	PaginationLimit   *uint64        `json:"paginationLimit,omitempty"`
}

type StoreMessageResponse struct {
	WakuMessage *wakuMessage `json:"message"`
	PubsubTopic string       `json:"pubsubTopic"`
	MessageHash MessageHash  `json:"messageHash"`
}

// UnmarshalJSON implements the json.Unmarshaler interface. The library wraps
// both message and pubsubTopic in Opt[T].
func (r *StoreMessageResponse) UnmarshalJSON(input []byte) error {
	var raw struct {
		WakuMessage json.RawMessage `json:"message"`
		PubsubTopic json.RawMessage `json:"pubsubTopic"`
		MessageHash MessageHash     `json:"messageHash"`
	}
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}

	r.MessageHash = raw.MessageHash

	if err := unmarshalOpt(raw.PubsubTopic, &r.PubsubTopic); err != nil {
		return err
	}

	if message := unwrapOpt(raw.WakuMessage); message != nil {
		r.WakuMessage = &wakuMessage{}
		if err := json.Unmarshal(message, r.WakuMessage); err != nil {
			return err
		}
	}

	return nil
}

type StoreQueryResponse struct {
	RequestId        string                  `json:"requestId,omitempty"`
	StatusCode       *uint32                 `json:"statusCode,omitempty"`
	StatusDesc       string                  `json:"statusDesc,omitempty"`
	Messages         *[]StoreMessageResponse `json:"messages,omitempty"`
	PaginationCursor MessageHash             `json:"paginationCursor,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface. The library wraps
// paginationCursor in Opt[T], which is empty on the last page.
func (r *StoreQueryResponse) UnmarshalJSON(input []byte) error {
	var raw struct {
		RequestId        string                  `json:"requestId,omitempty"`
		StatusCode       *uint32                 `json:"statusCode,omitempty"`
		StatusDesc       string                  `json:"statusDesc,omitempty"`
		Messages         *[]StoreMessageResponse `json:"messages,omitempty"`
		PaginationCursor json.RawMessage         `json:"paginationCursor,omitempty"`
	}
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}

	r.RequestId = raw.RequestId
	r.StatusCode = raw.StatusCode
	r.StatusDesc = raw.StatusDesc
	r.Messages = raw.Messages

	return unmarshalOpt(raw.PaginationCursor, &r.PaginationCursor)
}
