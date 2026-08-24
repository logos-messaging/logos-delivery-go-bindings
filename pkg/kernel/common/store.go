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

type StoreQueryResponse struct {
	RequestId        string                  `json:"requestId,omitempty"`
	StatusCode       *uint32                 `json:"statusCode,omitempty"`
	StatusDesc       string                  `json:"statusDesc,omitempty"`
	Messages         *[]StoreMessageResponse `json:"messages,omitempty"`
	PaginationCursor MessageHash             `json:"paginationCursor,omitempty"`
}

// UnmarshalJSON unwraps the Opt[T] fields the library wraps a stored message's
// content and pubsub topic in. An absent Opt leaves the field at its zero
// value.
func (r *StoreMessageResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		WakuMessage json.RawMessage `json:"message"`
		PubsubTopic json.RawMessage `json:"pubsubTopic"`
		MessageHash MessageHash     `json:"messageHash"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = StoreMessageResponse{MessageHash: raw.MessageHash}

	if value, ok := unwrapOpt(raw.PubsubTopic); ok {
		if err := json.Unmarshal(value, &r.PubsubTopic); err != nil {
			return err
		}
	}
	if value, ok := unwrapOpt(raw.WakuMessage); ok {
		var message wakuMessage
		if err := json.Unmarshal(value, &message); err != nil {
			return err
		}
		r.WakuMessage = &message
	}
	return nil
}

// UnmarshalJSON unwraps the Opt[T] the library wraps the pagination cursor in.
func (r *StoreQueryResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		RequestId        string                  `json:"requestId"`
		StatusCode       *uint32                 `json:"statusCode"`
		StatusDesc       string                  `json:"statusDesc"`
		Messages         *[]StoreMessageResponse `json:"messages"`
		PaginationCursor json.RawMessage         `json:"paginationCursor"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = StoreQueryResponse{
		RequestId:  raw.RequestId,
		StatusCode: raw.StatusCode,
		StatusDesc: raw.StatusDesc,
		Messages:   raw.Messages,
	}

	if value, ok := unwrapOpt(raw.PaginationCursor); ok {
		if err := json.Unmarshal(value, &r.PaginationCursor); err != nil {
			return err
		}
	}
	return nil
}

// Payload is the stored message's payload, or nil when the response carried no
// message content.
func (r *StoreMessageResponse) Payload() []byte {
	if r.WakuMessage == nil {
		return nil
	}
	return r.WakuMessage.Payload
}

// ContentTopic is the stored message's content topic, or "" when the response
// carried no message content.
func (r *StoreMessageResponse) ContentTopic() string {
	if r.WakuMessage == nil {
		return ""
	}
	return r.WakuMessage.ContentTopic
}
