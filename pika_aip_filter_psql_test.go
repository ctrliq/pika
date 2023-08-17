// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"testing"

	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestAIP160SimpleEquals(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	qs, err := qs.AIP160(`non_nullable = "String"`, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160DisallowNestedExpression(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	_, err := qs.AIP160(`(non_nullable = "String" AND (test = 1 AND (test = 2)))`, AIPFilterOptions{})
	require.NotNil(t, err)
	require.Equal(t, "nested expressions are not supported", err.Error())
}

func TestAIP160DisallowCombinatorValues(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	_, err := qs.AIP160(`non_nullable = ("String" OR "String2")`, AIPFilterOptions{})
	require.NotNil(t, err)
	require.Equal(t, "cannot combine multiple values in subexpression", err.Error())
}

func TestAIP160SimpleEqualsAcceptableIdentifier(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	qs, err := qs.AIP160(`non_nullable = "String"`, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	qs = Q[simpleModel3](psql)
	qs, err = qs.AIP160(`non_nullable = "String"`, AIPFilterOptions{
		AcceptableIdentifiers: []string{"non_nullable"},
	})
	require.Nil(t, err)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"String"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	qs = Q[simpleModel3](psql)
	_, err = qs.AIP160(`invalid = "String"`, AIPFilterOptions{
		AcceptableIdentifiers: []string{"non_nullable"},
	})
	require.NotNil(t, err)
	require.EqualError(t, err, "identifier invalid is not allowed")
}

func TestAIP160EqualsAndNull(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `non_nullable = "String" AND nullable = null`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1 AND "simpleModel3"."nullable" IS NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160NotSimple(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `NOT non_nullable = "String" AND nullable = null`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" != $1 AND "simpleModel3"."nullable" IS NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160NotCombined(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `NOT (non_nullable = "String" AND nullable = null)`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" != $1 AND "simpleModel3"."nullable" IS NOT NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160NotCombinedIsNull(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `-(non_nullable = "String" AND nullable = null) OR nullable = null`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" != $1 AND "simpleModel3"."nullable" IS NOT NULL) OR ("simpleModel3"."nullable" IS NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160NotInParens(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `(NOT non_nullable = "String" OR nullable = null) OR nullable = "String"`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" != $1 OR "simpleModel3"."nullable" IS NULL) OR ("simpleModel3"."nullable" = $2) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String", "String"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160EqualsAndNullCombined(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `(non_nullable = "String" AND nullable = null) OR num = 99999`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1 AND "simpleModel3"."nullable" IS NULL) OR ("simpleModel3"."num" = $2) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String", int64(99999)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160EqualsAndNullAndGreaterOrIDCombined(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `(non_nullable = "String" AND num > 1337) OR id = 1`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1 AND "simpleModel3"."num" > $2) OR ("simpleModel3"."id" = $3) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String", int64(1337), int64(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160EqualsOrNullAndGreaterOrIDCombined(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `(non_nullable = "String" OR num > 1337) OR id = 1`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1 OR "simpleModel3"."num" > $2) OR ("simpleModel3"."id" = $3) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String", int64(1337), int64(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160EqualsOrNullAndGreaterOrIDAndNullableCombined(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `(non_nullable = "String" OR num > 1337) OR id = 1 AND nullable = null OR (num = 1337 AND id = 2 OR nullable = null)`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1 OR "simpleModel3"."num" > $2) OR ("simpleModel3"."id" = $3) AND ("simpleModel3"."nullable" IS NULL) OR ("simpleModel3"."num" = $4 AND "simpleModel3"."id" = $5 OR "simpleModel3"."nullable" IS NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String", int64(1337), int64(1), int64(1337), int64(2)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160PresetFilters(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `non_nullable = "String"`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	args := orderedmap.New[string, any]()
	args.Set("id", 1)
	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" = $1) AND ("simpleModel3"."id" = $2) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{"String", 1}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160BoolFilters(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	filter := `bool = true AND bool_2 = true`

	qs, err := qs.AIP160(filter, AIPFilterOptions{})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."bool" = $1 AND "simpleModel3"."bool_2" = $2) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{true, true}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160HasArrayField(t *testing.T) {
	psql := newPsql(t)
	createTestModelWithArray(t, psql)
	qs := Q[simpleModelWithArray](psql)

	filter := `vars:"a"`

	qs, err := qs.AIP160(filter, AIPFilterOptions{
		Identifiers: map[string]AIPFilterIdentifier{
			"vars": {
				IsRepeated: true,
			},
		},
	})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModelWithArray"."id", "simpleModelWithArray"."vars", "simpleModelWithArray"."varsint32" FROM "simple_model_with_array" "simpleModelWithArray" WHERE ($1 = ANY("simpleModelWithArray"."vars"))`
	expectedArgs := []interface{}{"a"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}

func TestAIP160NotHasArrayField(t *testing.T) {
	psql := newPsql(t)
	createTestModelWithArray(t, psql)
	qs := Q[simpleModelWithArray](psql)

	filter := `-(vars:"a")`

	qs, err := qs.AIP160(filter, AIPFilterOptions{
		Identifiers: map[string]AIPFilterIdentifier{
			"vars": {
				IsRepeated: true,
			},
		},
	})
	require.Nil(t, err)

	expectedQuery := `SELECT "simpleModelWithArray"."id", "simpleModelWithArray"."vars", "simpleModelWithArray"."varsint32" FROM "simple_model_with_array" "simpleModelWithArray" WHERE ($1 != ALL("simpleModelWithArray"."vars"))`
	expectedArgs := []interface{}{"a"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)
}
