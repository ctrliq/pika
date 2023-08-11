// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type PageToken[T any] struct {
	QuerySet[T] `json:"-"`

	Offset   uint   `json:"offset"`
	Filter   string `json:"filter"`
	OrderBy  string `json:"order_by"`
	PageSize uint   `json:"page_size"`
}

type Paginatable interface {
	GetFilter() string
	GetOrderBy() string
	GetPageSize() int32
	GetPageToken() string
}

type PageRequest struct {
	// Filter is a filter expression that restricts the results to return.
	Filter string

	// OrderBy is a comma-separated list of fields to order by.
	OrderBy string

	// PageSize is the maximum number of results to return.
	PageSize int32

	// PageToken is the page token to use for the next request.
	PageToken string
}

func NewPageToken[T any]() *PageToken[T] {
	return &PageToken[T]{}
}

// Encode marshals the PageToken to a base64-encoded string
func (p *PageToken[T]) Encode() (string, error) {
	// Page token with page size 0 is useless
	if p.PageSize == 0 {
		return "", nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to marshal json: %w", err)
	}

	return base64.URLEncoding.EncodeToString(data), nil
}

// ParsePageToken constructs a PageToken from a base64-encoded string
func (p *PageToken[T]) Decode(s string) error {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("failed to decode page token, make sure it is from a previous request")
	}
	return json.Unmarshal(data, p)
}

func (p *PageToken[T]) pageToken(b QuerySet[T], options AIPFilterOptions) (QuerySet[T], error) {
	if p.OrderBy != "" {
		orderBy, err := options.verifyOrderBy(p.OrderBy)
		if err != nil {
			return nil, fmt.Errorf("applying order by from page token: %w (%s)", err, p.OrderBy)
		}
		b.OrderBy(orderBy...)
	}

	// Default page size is 25
	if p.PageSize == 0 {
		p.PageSize = 25
	}

	// Return error if page size is less than 1
	if p.PageSize < 1 {
		return nil, fmt.Errorf("page size cannot be less than 1")
	}

	// Return error if page size is greater than 100
	if p.PageSize > 100 {
		return nil, fmt.Errorf("page size cannot be greater than 100")
	}

	b.Offset(int(p.Offset))
	b.Limit(int(p.PageSize))

	if p.Filter != "" {
		_, err := b.AIP160(p.Filter, options)
		if err != nil {
			return nil, fmt.Errorf("applying filter from page token: %w", err)
		}
	}

	return b, nil
}

func (p *PageRequest) GetFilter() string {
	return p.Filter
}

func (p *PageRequest) GetOrderBy() string {
	return p.OrderBy
}

func (p *PageRequest) GetPageSize() int32 {
	return p.PageSize
}

func (p *PageRequest) GetPageToken() string {
	return p.PageToken
}
