// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var (
	PikaMetadataTableName      = "PikaTableName"
	PikaMetadataDefaultOrderBy = "PikaDefaultOrderBy"
	PikaMetadataFields         = []string{
		PikaMetadataTableName,
		PikaMetadataDefaultOrderBy,
	}
)

func Q[T any](x any) QuerySet[T] {
	if sql, ok := x.(*PostgreSQL); ok {
		return PSQLQuery[T](sql)
	}

	panic("unsupported database")
}

// QuerySet is a chainable interface for building queries.
// WARNING: This interface is not thread-safe and is meant to be used in a single goroutine.
// Do not re-use a QuerySet after calling All, Get, GetOrNil, Count, or any other method that returns a value.
// Creating a QuerySet is cheap and is meant to be discarded after use.
// It builds up a query string and arguments, and then executes the query.
// The query set can be passed onto another goroutine, given that it is modified only by one party.
// For safety, DO NOT pass a QuerySet to another goroutine and always create a new one.
// Use Q to create a new QuerySet.
//
//nolint:interfacebloat
type QuerySet[T any] interface {
	// Filter returns a new QuerySet with the given filters applied.
	// The filters are applied in the order they are given.
	// Only use named parameters in the filters.
	// Multiple filter calls can be made, they will be combined with AND.
	// Will also work as AND combined
	// See FilterOr for OR combined.
	// See FilterInnerOr for inner filters combined with OR.
	// See FilterOrInnerOr for inner filters combined with OR.
	// Filter keys can also contain various hints (use as suffix to filter key):
	//  - "__ne" to negate the filter
	//  - "__in" to use an IN clause
	//  - "__nin" to use a NOT IN clause
	//  - "__gt" to use a > clause
	//  - "__gte" to use a >= clause
	//  - "__lt" to use a < clause
	//  - "__lte" to use a <= clause
	//  - "__like" to use a LIKE clause
	//  - "__nlike" to use a NOT LIKE clause
	//  - "__ilike" to use a ILIKE clause
	//  - "__nilike" to use a NOT ILIKE clause
	//  - "__null" to use a IS NULL clause
	//  - "__notnull" to use a IS NOT NULL clause
	//  - "__or" to prepend with OR instead of AND (in AND filter calls)
	//  - "__and" to prepend with AND instead of OR (in OR filter calls)
	Filter(queries ...string) QuerySet[T]

	// FilterOr returns a new QuerySet with the given filters applied.
	// The filters are applied in the order they are given.
	// Only use named parameters in the filters.
	// Multiple filter calls can be made, they will be combined with AND.
	// But will work as OR combined
	// See Filter for AND combined.
	FilterOr(queries ...string) QuerySet[T]

	// FilterInnerOr returns a new QuerySet with the given filters applied.
	// Same as Filter, but inner filters are combined with OR.
	FilterInnerOr(queries ...string) QuerySet[T]

	// FilterOrInnerOr returns a new QuerySet with the given filters applied.
	// Same as FilterOr, but inner filters are combined with OR.
	FilterOrInnerOr(queries ...string) QuerySet[T]

	// Args sets named arguments for the filters.
	// The arguments are applied in the order they are given.
	Args(args *orderedmap.OrderedMap[string, interface{}]) QuerySet[T]

	// ClearArgs clear filters, args, and any previous set joins
	ClearAll() QuerySet[T]

	// Create creates a new value
	Create(value *T) error

	// Update updates a value
	// All filters will be applied
	Update(value *T) error

	// Delete deletes a row
	// All filters will be applied
	Delete() error

	// GetOrNil returns a single value or nil
	// Multiple values will return an error.
	// Ignores Limit
	GetOrNil() (*T, error)

	// Get returns a single value
	// Returns error if no value is found
	// Returns error if multiple values are found
	// Ignores Limit
	Get() (*T, error)

	// All returns all values
	All() ([]*T, error)

	// Count returns the number of values
	Count() (int, error)

	// Limit sets the limit for the query
	Limit(limit int) QuerySet[T]

	// Offset sets the offset for the query
	Offset(offset int) QuerySet[T]

	// OrderBy sets the order for the query
	// Use - to indicate descending order
	// Example:
	// 	OrderBy("-id", "name")
	OrderBy(order ...string) QuerySet[T]

	// ResetOrderBy resets the order for the query
	ResetOrderBy() QuerySet[T]

	// Query related methods

	// CreateQuery returns the query and args for Create
	CreateQuery(value *T) (string, []interface{})

	// UpdateQuery returns the query and args for Update
	UpdateQuery(value *T) (string, []interface{})

	// DeleteQuery returns the query and args for Delete
	DeleteQuery() (string, []interface{})

	// GetOrNilQuery returns the query and args for GetOrNil
	GetOrNilQuery() (string, []interface{})

	// GetQuery returns the query and args for Get
	GetQuery() (string, []interface{})

	// AllQuery returns the query and args for All
	AllQuery() (string, []interface{})

	// Extensions
	// AIP-160 filtering for gRPC/Proto
	// See https://google.aip.dev/160
	AIP160(filter string, options AIPFilterOptions) (QuerySet[T], error)

	// Page token functionality for gRPC
	GetPage(paginatable Paginatable, options AIPFilterOptions) ([]*T, string, error)

	// Join table
	InnerJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]
	LeftJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]
	RightJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]
	FullJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]

	// Exclude fields
	Exclude(excludes ...string) QuerySet[T]
	// Include fields
	Include(includes ...string) QuerySet[T]
}

func NewArgs() *orderedmap.OrderedMap[string, any] {
	return orderedmap.New[string, any]()
}
