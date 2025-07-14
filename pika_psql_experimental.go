// SPDX-FileCopyrightText: Copyright (c) 2023-2025, CTRL IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// Static errors for err113 compliance
var (
	ErrIDNotFound = errors.New("id not found")
)

func (b *basePsql[T]) findID(x *T) any {
	elem := reflect.ValueOf(x).Elem()

	// Check if ID is a field
	idField := elem.FieldByName("ID")
	if idField.IsValid() {
		return idField.Interface()
	}

	// Also check for Id field
	idField = elem.FieldByName("Id")
	if idField.IsValid() {
		return idField.Interface()
	}

	// Return nil if ID is not a field
	return nil
}

func (b *basePsql[T]) F(keyval ...any) QuerySet[T] {
	args := NewArgs()
	var queries []string
	for i := 0; i < len(keyval); i += 2 {
		args.Set(keyval[i].(string), keyval[i+1])
		filter := fmt.Sprintf("%s=:%s", keyval[i].(string), keyval[i].(string))
		queries = append(queries, filter)
	}

	logger.Debugf("F: %s", queries)

	return b.Args(args).Filter(queries...)
}

func (b *basePsql[T]) D(ctx context.Context, x *T) error {
	id := b.findID(x)
	if id == nil {
		return ErrIDNotFound
	}

	qs := b.F("id", id)
	return qs.Delete(ctx)
}

func (b *basePsql[T]) Transaction(ctx context.Context) (QuerySet[T], error) {
	ts := NewPostgreSQLFromDB(b.psql.DB())
	err := ts.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return Q[T](ts), nil
}

func (b *basePsql[T]) U(ctx context.Context, x *T) error {
	id := b.findID(x)
	if id == nil {
		return ErrIDNotFound
	}

	qs := b.F("id", id)
	return qs.Update(ctx, x)
}
