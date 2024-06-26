// SPDX-FileCopyrightText: Copyright (c) 2023-2024, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"context"

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
	Create(ctx context.Context, value *T, options ...CreateOption) error

	// Update updates a value
	// All filters will be applied
	Update(ctx context.Context, value *T) error

	// Delete deletes a row
	// All filters will be applied
	Delete(ctx context.Context) error

	// GetOrNil returns a single value or nil
	// Multiple values will return an error.
	// Ignores Limit
	GetOrNil(ctx context.Context) (*T, error)

	// Get returns a single value
	// Returns error if no value is found
	// Returns error if multiple values are found
	// Ignores Limit
	Get(ctx context.Context) (*T, error)

	// All returns all values
	All(ctx context.Context) ([]*T, error)

	// Count returns the number of values
	Count(ctx context.Context) (int, error)

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
	CreateQuery(value *T, options ...CreateOption) (string, []interface{})

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
	// The count is optional and returns the total number of rows for the query.
	// It is implemented as a variadic function to not break existing code.
	GetPage(ctx context.Context, paginatable Paginatable, options AIPFilterOptions, count ...*int) ([]*T, string, error)

	// Join table
	InnerJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]
	LeftJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]
	RightJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]
	FullJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T]

	// Exclude fields
	Exclude(excludes ...string) QuerySet[T]
	// Include fields
	Include(includes ...string) QuerySet[T]

	// EXPERIMENTAL
	// The following methods are EXPERIMENTAL. Think of it as a sneak peek on what's coming.
	// It is mostly to experiment with a simpler API for filtering, updating and querying.
	// Feel free to test it out and provide feedback.

	// U is a shorthand for Update. ID field is used as the filter.
	// Other filters applied to the query set are also inherited.
	// Returns an error if the ID field is not set or does not exist.
	// Thus preventing accidental updates to all rows.
	U(ctx context.Context, value *T) error

	// F is a shorthand for Filter. It is a variadic function that accepts a list of filters.
	// The filters are applied in the order they are given.
	// Format is as follows: <KEY>, <VALUE> etc.
	F(keyval ...any) QuerySet[T]

	// D is a shorthand for Delete. ID field is used as the filter.
	// Other filters applied to the query set are also inherited.
	// Returns an error if the ID field is not set or does not exist.
	// Thus preventing accidental deletes to all rows.
	D(ctx context.Context, value *T) error

	// Transaction is a shorthand for wrapping a query set in a transaction.
	// Currently Pika transactions affects the full connection, not just the query set.
	// That method works if you use factories to create query sets.
	// This helper will re-use the internal DB instance to return a new query set with the transaction.
	Transaction(ctx context.Context) (QuerySet[T], error)
}

func NewArgs() *orderedmap.OrderedMap[string, any] {
	return orderedmap.New[string, any]()
}
