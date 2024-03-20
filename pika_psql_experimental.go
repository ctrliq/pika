// SPDX-FileCopyrightText: Copyright (c) 2023-2024, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"context"
	"fmt"
	"reflect"
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

func (b *basePsql[T]) D(x *T) error {
	id := b.findID(x)
	if id == nil {
		return fmt.Errorf("id not found")
	}

	qs := b.F("id", id)
	return qs.Delete()
}

func (b *basePsql[T]) Transaction(ctx context.Context) (QuerySet[T], error) {
	ts := NewPostgreSQLFromDB(b.psql.DB())
	err := ts.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return Q[T](ts), nil
}

func (b *basePsql[T]) U(x *T) error {
	id := b.findID(x)
	if id == nil {
		return fmt.Errorf("id not found")
	}

	qs := b.F("id", id)
	return qs.Update(x)
}
