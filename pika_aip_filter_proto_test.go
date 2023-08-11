// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/stretchr/testify/require"
	"go.ciq.dev/pika/parser"
	pikatestpb "go.ciq.dev/pika/testproto"
)

type protoModel4 struct {
	PikaTableName string `pika:"proto_model_4"`

	ID           int            `db:"id"`
	NullableInt  sql.NullInt64  `db:"nullable_int"`
	NullableBool sql.NullBool   `db:"nullable_bool"`
	Bool         bool           `db:"bool"`
	Timestamp    sql.NullTime   `db:"timestamp"`
	Status       int            `db:"status"`
	Strs         pq.StringArray `db:"strs"`
}

func createTestEntries4(t *testing.T, psql *PostgreSQL) {
	_, err := psql.db.Exec("DROP TABLE IF EXISTS proto_model_4")
	require.Nil(t, err)

	_, err = psql.db.Exec("CREATE TABLE proto_model_4 (id SERIAL PRIMARY KEY, nullable_int NUMERIC, nullable_bool BOOL, \"bool\" BOOL NOT NULL, timestamp TIMESTAMPTZ, status INT, strs TEXT[])")
	require.Nil(t, err)

	_, err = psql.db.Exec(`
		INSERT INTO proto_model_4 (id, nullable_int, nullable_bool, "bool", timestamp, status, strs)
		VALUES
		(1, 1, true, true, '2020-01-01 00:00:00', 1, '{a}'),
		(2, 2, false, false, '2020-01-01 00:00:00', 2, '{b}'),
		(3, 100, NULL, false, NULL, 1, '{c}'),
		(4, NULL, NULL, true, NULL, 2, '{d}')
	`)
	require.Nil(t, err)
}

func TestSimple1ProtoReflect(t *testing.T) {
	x := &pikatestpb.Simple1{}

	opts := ProtoReflect(x)
	require.Len(t, opts.AcceptableIdentifiers, 1)
	require.Equal(t, "name", opts.AcceptableIdentifiers[0])
	require.Len(t, opts.Identifiers, 1)
	require.Contains(t, opts.Identifiers, "name")
	require.Len(t, opts.Identifiers["name"].AcceptedTypes, 1)
	require.Equal(t, parser.FilterLexerSTRING, opts.Identifiers["name"].AcceptedTypes[0])
}

func TestSimpleWrappers2ProtoReflect(t *testing.T) {
	x := &pikatestpb.SimpleWrappers2{}

	opts := ProtoReflect(x)
	require.Len(t, opts.AcceptableIdentifiers, 1)
	require.Equal(t, "name", opts.AcceptableIdentifiers[0])
	require.Len(t, opts.Identifiers, 1)
	require.Contains(t, opts.Identifiers, "name")
	require.Len(t, opts.Identifiers["name"].AcceptedTypes, 2)
	require.Equal(t, parser.FilterLexerSTRING, opts.Identifiers["name"].AcceptedTypes[0])
	require.Equal(t, parser.FilterLexerNULL, opts.Identifiers["name"].AcceptedTypes[1])
}

func TestComplete3ProtoReflect(t *testing.T) {
	x := &pikatestpb.Complete3{}

	opts := ProtoReflect(x)
	require.Len(t, opts.AcceptableIdentifiers, 7)
	require.Equal(t, "str", opts.AcceptableIdentifiers[0])
	require.Equal(t, "nullableInt", opts.AcceptableIdentifiers[1])
	require.Equal(t, "nullableBool", opts.AcceptableIdentifiers[2])
	require.Equal(t, "bool", opts.AcceptableIdentifiers[3])
	require.Equal(t, "timestamp", opts.AcceptableIdentifiers[4])
	require.Equal(t, "status", opts.AcceptableIdentifiers[5])
	require.Equal(t, "strs", opts.AcceptableIdentifiers[6])
	require.Len(t, opts.Identifiers, 7)
	require.Contains(t, opts.Identifiers, "str")
	require.Contains(t, opts.Identifiers, "nullableInt")
	require.Contains(t, opts.Identifiers, "nullableBool")
	require.Contains(t, opts.Identifiers, "bool")
	require.Contains(t, opts.Identifiers, "timestamp")
	require.Contains(t, opts.Identifiers, "status")
	require.Contains(t, opts.Identifiers, "strs")
	require.Len(t, opts.Identifiers["str"].AcceptedTypes, 1)
	require.Equal(t, parser.FilterLexerSTRING, opts.Identifiers["str"].AcceptedTypes[0])
	require.Len(t, opts.Identifiers["nullableInt"].AcceptedTypes, 2)
	require.Equal(t, parser.FilterLexerNUM_INT, opts.Identifiers["nullableInt"].AcceptedTypes[0])
	require.Equal(t, parser.FilterLexerNULL, opts.Identifiers["nullableInt"].AcceptedTypes[1])
	require.Len(t, opts.Identifiers["nullableBool"].AcceptedTypes, 3)
	require.Equal(t, parser.FilterLexerTRUE, opts.Identifiers["nullableBool"].AcceptedTypes[0])
	require.Equal(t, parser.FilterLexerFALSE, opts.Identifiers["nullableBool"].AcceptedTypes[1])
	require.Equal(t, parser.FilterLexerNULL, opts.Identifiers["nullableBool"].AcceptedTypes[2])
	require.Len(t, opts.Identifiers["bool"].AcceptedTypes, 2)
	require.Equal(t, parser.FilterLexerTRUE, opts.Identifiers["bool"].AcceptedTypes[0])
	require.Equal(t, parser.FilterLexerFALSE, opts.Identifiers["bool"].AcceptedTypes[1])
	require.Len(t, opts.Identifiers["timestamp"].AcceptedTypes, 2)
	require.Equal(t, parser.FilterLexerTIMESTAMP, opts.Identifiers["timestamp"].AcceptedTypes[0])
	require.Equal(t, parser.FilterLexerNULL, opts.Identifiers["timestamp"].AcceptedTypes[1])
	require.Len(t, opts.Identifiers["status"].AcceptedTypes, 1)
	require.Equal(t, parser.FilterLexerNUM_INT, opts.Identifiers["status"].AcceptedTypes[0])
	require.Len(t, opts.Identifiers["status"].ValueAliases, 14)
	require.Equal(t, int64(0), opts.Identifiers["status"].ValueAliases["unspecified"])
	require.Equal(t, int64(0), opts.Identifiers["status"].ValueAliases["status_unspecified"])
	require.Equal(t, int64(1), opts.Identifiers["status"].ValueAliases["ok"])
	require.Equal(t, int64(1), opts.Identifiers["status"].ValueAliases["status_ok"])
	require.Equal(t, int64(2), opts.Identifiers["status"].ValueAliases["error"])
	require.Equal(t, int64(2), opts.Identifiers["status"].ValueAliases["status_error"])
	require.Len(t, opts.Identifiers["strs"].AcceptedTypes, 1)
	require.Equal(t, parser.FilterLexerSTRING, opts.Identifiers["strs"].AcceptedTypes[0])
	require.True(t, opts.Identifiers["strs"].IsRepeated)
}

func TestComplete3AIP160(t *testing.T) {
	opts := ProtoReflect(&pikatestpb.Complete3{})

	psql := newPsql(t)
	createTestEntries4(t, psql)

	filter := `(bool = true AND timestamp = null) AND status = 1`
	qs := Q[protoModel4](psql)
	qs, err := qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery := `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ("protoModel4"."bool" = $1 AND "protoModel4"."timestamp" IS NULL) AND ("protoModel4"."status" = $2)`
	expectedArgs := []interface{}{true, int64(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ts := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	filter = fmt.Sprintf(`(timestamp = %s OR timestamp = null) AND status = 1`, ts.Format(time.RFC3339))
	qs = Q[protoModel4](psql)
	qs, err = qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery = `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ("protoModel4"."timestamp" = $1 OR "protoModel4"."timestamp" IS NULL) AND ("protoModel4"."status" = $2)`
	expectedArgs = []interface{}{ts, int64(1)}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	filter = `non_existent = 1`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.NotNil(t, err)
	require.Equal(t, "identifier non_existent is not allowed", err.Error())

	filter = `bool = null`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.NotNil(t, err)
	require.Equal(t, "type NULL is not accepted for identifier bool", err.Error())

	filter = `status = 0`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.Nil(t, err)

	filter = `status = "OK"`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery = `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ("protoModel4"."status" = $1)`
	expectedArgs = []interface{}{int64(1)}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	filter = `status = "status_ok"`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery = `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ("protoModel4"."status" = $1)`
	expectedArgs = []interface{}{int64(1)}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	filter = `status = "invalid"`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.NotNil(t, err)
	require.Equal(t, "type STRING is not accepted for identifier status", err.Error())

	filter = `status = 99`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.NotNil(t, err)
	require.Equal(t, "value 99 is not accepted for identifier status", err.Error())
}

func TestComplete3Exclude(t *testing.T) {
	opts := ProtoReflectWithOpts(&pikatestpb.Complete3{}, ProtoReflectOptions{
		Exclude: []string{"nullable_int", "nullable_bool"},
	})

	psql := newPsql(t)
	createTestEntries4(t, psql)

	filter := `(bool = true AND timestamp = null) AND status = 1`
	qs := Q[protoModel4](psql)
	qs, err := qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery := `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ("protoModel4"."bool" = $1 AND "protoModel4"."timestamp" IS NULL) AND ("protoModel4"."status" = $2)`
	expectedArgs := []interface{}{true, int64(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	filter = `nullable_int = 1 AND bool = false`
	qs = Q[protoModel4](psql)
	_, err = qs.AIP160(filter, opts)
	require.NotNil(t, err)
	require.Equal(t, "identifier nullable_int is not allowed", err.Error())
}

func TestMultipleSameFieldOr(t *testing.T) {
	opts := ProtoReflectWithOpts(&pikatestpb.Complete3{}, ProtoReflectOptions{})

	psql := newPsql(t)
	createTestEntries4(t, psql)

	filter := `status = 1 OR status = 2 OR status = 3 OR status = 4 OR status = 5 OR status = 6`
	qs := Q[protoModel4](psql)
	qs, err := qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery := `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ("protoModel4"."status" = $1 OR "protoModel4"."status" = $2 OR "protoModel4"."status" = $3 OR "protoModel4"."status" = $4 OR "protoModel4"."status" = $5 OR "protoModel4"."status" = $6)`
	expectedArgs := []interface{}{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestArrayHas(t *testing.T) {
	opts := ProtoReflectWithOpts(&pikatestpb.Complete3{}, ProtoReflectOptions{})

	psql := newPsql(t)
	createTestEntries4(t, psql)

	filter := `strs:"a"`
	qs := Q[protoModel4](psql)
	qs, err := qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery := `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ($1 = ANY("protoModel4"."strs"))`
	expectedArgs := []interface{}{"a"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestArrayNotHas(t *testing.T) {
	opts := ProtoReflectWithOpts(&pikatestpb.Complete3{}, ProtoReflectOptions{})

	psql := newPsql(t)
	createTestEntries4(t, psql)

	filter := `-(strs:"a")`
	qs := Q[protoModel4](psql)
	qs, err := qs.AIP160(filter, opts)
	require.Nil(t, err)

	expectedQuery := `SELECT "protoModel4"."id", "protoModel4"."nullable_int", "protoModel4"."nullable_bool", "protoModel4"."bool", "protoModel4"."timestamp", "protoModel4"."status", "protoModel4"."strs" FROM "proto_model_4" "protoModel4" WHERE ($1 != ALL("protoModel4"."strs"))`
	expectedArgs := []interface{}{"a"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}
