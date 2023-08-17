// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var pgInstance *embeddedpostgres.EmbeddedPostgres

type simpleModel1 struct {
	PikaTableName string `pika:"simple_model_1"`

	ID          int    `db:"id"`
	Title       string `db:"title"`
	Description string `db:"description"`

	ExcludeMe string `db:"-"`
}

type simpleModel2 struct {
	PikaTableName      string `pika:"simple_model_2"`
	PikaDefaultOrderBy string `pika:"-created_at"`

	ID        int        `db:"id"`
	CreatedAt *time.Time `db:"created_at"`
	Name      string     `db:"name"`
}

type simpleModel3 struct {
	PikaTableName      string `pika:"simple_model_3"`
	PikaDefaultOrderBy string `pika:"id"`

	ID          int            `db:"id"`
	Num         int            `db:"num"`
	NonNullable string         `db:"non_nullable"`
	Nullable    sql.NullString `db:"nullable"`
}

type simpleModelCreate struct {
	PikaTableName string `pika:"simple_model_create"`

	ID          int    `db:"id" pika:"omitempty"`
	Title       string `db:"title"`
	Description string `db:"description"`
}

type simpleModelWithArray struct {
	PikaTableName string `pika:"simple_model_with_array"`

	ID        int            `db:"id" pika:"omitempty"`
	Vars      pq.StringArray `db:"vars"`
	VarsInt32 pq.Int32Array  `db:"varsint32"`
}

type noExplicitTableName struct {
	ID int `db:"id"`
}

type joinSimpleModelMain struct {
	PikaTableName string `pika:"join_simple_model_main"`
	ID            int    `db:"id"`
	Name          string `db:"name"`
	ReferId       int    `db:"id" pika:"join_model_foreign.id"`
}

type joinComplexModelMain struct {
	PikaTableName  string `pika:"join_complex_model_main"`
	ID             int    `db:"id"`
	Name           string `db:"name"`
	ReferId        int    `db:"id" pika:"join_model_foreign.id"`
	AnotherReferId int    `db:"id" pika:"join_model_another_foreign.id"`
}

type joinModelForeign struct {
	PikaTableName string `pika:"join_model_foreign"`
	ID            int    `db:"id"`
	ForeignKey    int    `db:"foreign_key"`
}

type joinModelAnotherForeign struct {
	PikaTableName string `pika:"join_model_another_foreign"`
	ID            int    `db:"id"`
	ForeignKey    int    `db:"foreign_key"`
	Product       string `db:"product"`
}

func newPsql(t *testing.T) *PostgreSQL {
	dbName := "postgres"
	port := 45111
	psql, err := NewPostgreSQL(fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", port, dbName))
	require.Nil(t, err)

	return psql
}

func newPsqlQuery[T any](t *testing.T) QuerySet[T] {
	qs := PSQLQuery[T](newPsql(t))

	return qs
}

func createTestEntries(t *testing.T, psql *PostgreSQL) {
	_, err := psql.db.Exec("DROP TABLE IF EXISTS simple_model_1")
	require.Nil(t, err)

	_, err = psql.db.Exec("CREATE TABLE simple_model_1 (id SERIAL PRIMARY KEY, title TEXT, description TEXT)")
	require.Nil(t, err)

	_, err = psql.db.Exec(`
		INSERT INTO simple_model_1 (id, title, description)
		VALUES
		(1, 'Test', 'Test'),
		(2, 'Test2', 'Test2'),
		(3, 'Test3', 'Test3')
	`)
	require.Nil(t, err)
}

func createTestEntries2(t *testing.T, psql *PostgreSQL) {
	_, err := psql.db.Exec("DROP TABLE IF EXISTS simple_model_2")
	require.Nil(t, err)

	_, err = psql.db.Exec("CREATE TABLE simple_model_2 (id SERIAL PRIMARY KEY, created_at TIMESTAMPTZ DEFAULT NOW(), name TEXT)")
	require.Nil(t, err)

	_, err = psql.db.Exec(`
		INSERT INTO simple_model_2 (id, created_at, name)
		VALUES
		(1, '2021-06-01', 'NewEmployee'),
		(2, '2021-06-21', 'NewDistro'),
		(3, '2022-07-14', 'NewVersion')
	`)
	require.Nil(t, err)
}

func createTestEntries3(t *testing.T, psql *PostgreSQL) {
	_, err := psql.db.Exec("DROP TABLE IF EXISTS simple_model_3")
	require.Nil(t, err)

	_, err = psql.db.Exec("CREATE TABLE simple_model_3 (id SERIAL PRIMARY KEY, num NUMERIC NOT NULL, non_nullable TEXT NOT NULL, nullable text)")
	require.Nil(t, err)

	_, err = psql.db.Exec(`
		INSERT INTO simple_model_3 (id, num, non_nullable, nullable)
		VALUES
		(1, 1, 'This is a longer string, MATCH THIS, please', null),
		(2, 1337, 'String', 'Not null'),
		(3, 99999, 'String', null)
	`)
	require.Nil(t, err)
}

func createTestModelCreate(t *testing.T, psql *PostgreSQL) {
	_, err := psql.db.Exec("DROP TABLE IF EXISTS simple_model_create")
	require.Nil(t, err)

	_, err = psql.db.Exec("CREATE TABLE simple_model_create (id SERIAL PRIMARY KEY, title TEXT, description TEXT)")
	require.Nil(t, err)
}

func createTestModelWithArray(t *testing.T, psql *PostgreSQL) {
	_, err := psql.db.Exec("DROP TABLE IF EXISTS simple_model_with_array")
	require.Nil(t, err)

	_, err = psql.db.Exec("CREATE TABLE simple_model_with_array (id SERIAL PRIMARY KEY, vars TEXT[] NOT NULL, varsint32 NUMERIC[] NOT NULL)")
	require.Nil(t, err)

	_, err = psql.db.Exec(`
		INSERT INTO simple_model_with_array (id, vars, varsint32)
		VALUES
		(1, '{a,b,c}', '{1,2,3}'),
		(2, '{d,e,f}', '{4,5,6}'),
		(3, '{g,h,i}', '{7,8,9}')
	`)
	require.Nil(t, err)
}

func createSimpleJoinModel(t *testing.T, psql *PostgreSQL) {
	psql.db.MustExec("DROP TABLE IF EXISTS join_model_foreign")
	psql.db.MustExec("DROP TABLE IF EXISTS join_simple_model_main")

	psql.db.MustExec("CREATE TABLE join_simple_model_main (id SERIAL PRIMARY KEY, name TEXT, description TEXT)")
	psql.db.MustExec("CREATE TABLE join_model_foreign (id SERIAL PRIMARY KEY, foreign_key SERIAL, CONSTRAINT fk_refer_id FOREIGN KEY (foreign_key) REFERENCES join_simple_model_main (id))")

	psql.db.MustExec(`INSERT INTO join_simple_model_main VALUES ($1, $2)`, int32(1), "jason")
	psql.db.MustExec(`INSERT INTO join_model_foreign VALUES ($1, $2)`, int32(1), int32(1))
}

func createComplexJoinModel(t *testing.T, psql *PostgreSQL) {
	psql.db.MustExec("DROP TABLE IF EXISTS join_model_foreign")
	psql.db.MustExec("DROP TABLE IF EXISTS join_model_another_foreign")
	psql.db.MustExec("DROP TABLE IF EXISTS join_complex_model_main")

	psql.db.MustExec("CREATE TABLE join_complex_model_main (id SERIAL PRIMARY KEY, name TEXT, description TEXT)")
	psql.db.MustExec("CREATE TABLE join_model_foreign (id SERIAL PRIMARY KEY, foreign_key SERIAL, CONSTRAINT fk_refer_id FOREIGN KEY (foreign_key) REFERENCES join_complex_model_main (id))")
	psql.db.MustExec("CREATE TABLE join_model_another_foreign (id SERIAL PRIMARY KEY, foreign_key SERIAL, product TEXT, CONSTRAINT fk_refer_id FOREIGN KEY (foreign_key) REFERENCES join_complex_model_main (id))")

	psql.db.MustExec(`INSERT INTO join_complex_model_main VALUES ($1, $2)`, int32(1), "jason")
	psql.db.MustExec(`INSERT INTO join_model_foreign VALUES ($1, $2)`, int32(1), int32(1))
	psql.db.MustExec(`INSERT INTO join_model_another_foreign VALUES ($1, $2, $3)`, int32(1), int32(1), "product")
}

func getMockBasePsql(t *testing.T) *basePsql[simpleModel1] {
	return PSQLQuery[simpleModel1](newPsql(t)).(*basePsql[simpleModel1])
}

func TestMain(m *testing.M) {
	var tempDir string
	var err error
	tempDir, err = os.MkdirTemp("", "pika")
	if err != nil {
		panic(err)
	}

	database := embeddedpostgres.NewDatabase(
		embeddedpostgres.
			DefaultConfig().
			Port(45111).
			DataPath(filepath.Join(tempDir, "data")).
			RuntimePath(filepath.Join(tempDir, "runtime")),
	)
	if err := database.Start(); err != nil {
		panic(err)
	}

	pgInstance = database

	exitCode := m.Run()

	if pgInstance != nil {
		if err := pgInstance.Stop(); err != nil {
			panic(err)
		}
		_ = os.RemoveAll(tempDir)
	}

	os.Exit(exitCode)
}

func TestPsqlSelectList(t *testing.T) {
	str := getMockBasePsql(t).psqlSelectList(nil, nil, false)

	require.Equal(t, "SELECT \"simpleModel1\".\"id\", \"simpleModel1\".\"title\", \"simpleModel1\".\"description\" FROM \"simple_model_1\" \"simpleModel1\"", str)
}

func TestPsqlSelectListExcludeColumn(t *testing.T) {
	str := getMockBasePsql(t).psqlSelectList([]string{"title"}, nil, false)

	require.Equal(t, "SELECT \"simpleModel1\".\"id\", \"simpleModel1\".\"description\" FROM \"simple_model_1\" \"simpleModel1\"", str)
}

func TestPsqlSelectListExcludeIncludeColumn(t *testing.T) {
	str := getMockBasePsql(t).psqlSelectList([]string{"title"}, []string{"id", "title"}, false)

	require.Equal(t, "SELECT \"simpleModel1\".\"id\" FROM \"simple_model_1\" \"simpleModel1\"", str)
}

func TestNoExplicitTableNamePlural(t *testing.T) {
	qs := newPsqlQuery[noExplicitTableName](t)

	expectedQuery := `SELECT "noExplicitTableName"."id" FROM "no_explicit_table_names" "noExplicitTableName" LIMIT 1`
	actualQuery, _ := qs.GetQuery()
	require.Equal(t, expectedQuery, actualQuery)
}

func TestNoExplicitTableNameAlias(t *testing.T) {
	psql := newPsql(t)
	psql.TableAlias("noExplicitTableName", "alias_table")

	qs := PSQLQuery[noExplicitTableName](psql)

	expectedQuery := `SELECT "noExplicitTableName"."id" FROM "alias_table" "noExplicitTableName" LIMIT 1`
	actualQuery, _ := qs.GetQuery()
	require.Equal(t, expectedQuery, actualQuery)
}

func TestGetOrNil(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)

	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1) LIMIT 1`
	expectedArgs := []interface{}{1}
	actualQuery, actualArgs := qs.GetOrNilQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.GetOrNil()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 1, ret.ID)
	require.Equal(t, "Test", ret.Title)
	require.Equal(t, "Test", ret.Description)
}

func TestGetOrNilNotFound(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 999)
	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1) LIMIT 1`
	expectedArgs := []interface{}{999}
	actualQuery, actualArgs := qs.GetOrNilQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.GetOrNil()
	require.Nil(t, err)
	require.Nil(t, ret)
}

func TestGetOrNilOrFilter(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	args.Set("id2", 2)
	qs = qs.
		Filter("id=:id").
		FilterOr("id=:id2").
		Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1) OR ("simpleModel1"."id" = $2) LIMIT 1`
	expectedArgs := []interface{}{1, 2}
	actualQuery, actualArgs := qs.GetOrNilQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.GetOrNil()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 1, ret.ID)
	require.Equal(t, "Test", ret.Title)
	require.Equal(t, "Test", ret.Description)
}

func TestGet(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1) LIMIT 1`
	expectedArgs := []interface{}{1}
	actualQuery, actualArgs := qs.GetQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.Get()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 1, ret.ID)
	require.Equal(t, "Test", ret.Title)
	require.Equal(t, "Test", ret.Description)
}

func TestGetNotFound(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 999)
	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1) LIMIT 1`
	expectedArgs := []interface{}{999}
	actualQuery, actualArgs := qs.GetQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.Get()
	require.NotNil(t, err)
	require.EqualError(t, err, "sql: no rows in result set")
	require.Nil(t, ret)
}

func TestAll(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1"`
	expectedArgs := []interface{}{}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 3, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, "Test", ret[0].Title)
	require.Equal(t, "Test", ret[0].Description)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "Test2", ret[1].Title)
	require.Equal(t, "Test2", ret[1].Description)
	require.Equal(t, 3, ret[2].ID)
	require.Equal(t, "Test3", ret[2].Title)
	require.Equal(t, "Test3", ret[2].Description)
}

func TestAllWithFilter(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1)`
	expectedArgs := []interface{}{1}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, "Test", ret[0].Title)
	require.Equal(t, "Test", ret[0].Description)
}

func TestAllWithFilterOr(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	args.Set("id2", 2)
	qs = qs.
		Filter("id=:id").
		FilterOr("id=:id2").
		Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1) OR ("simpleModel1"."id" = $2)`
	expectedArgs := []interface{}{1, 2}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, "Test", ret[0].Title)
	require.Equal(t, "Test", ret[0].Description)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "Test2", ret[1].Title)
	require.Equal(t, "Test2", ret[1].Description)
}

func TestAllLimit(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	qs = qs.Limit(2)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" LIMIT 2`
	expectedArgs := []interface{}{}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, "Test", ret[0].Title)
	require.Equal(t, "Test", ret[0].Description)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "Test2", ret[1].Title)
	require.Equal(t, "Test2", ret[1].Description)
}

func TestAllLimitOrderBy(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	qs = qs.Limit(2).OrderBy("id")

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" ORDER BY "simpleModel1"."id" ASC LIMIT 2`
	expectedArgs := []interface{}{}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, "Test", ret[0].Title)
	require.Equal(t, "Test", ret[0].Description)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "Test2", ret[1].Title)
	require.Equal(t, "Test2", ret[1].Description)
}

func TestAllLimitOrderByDesc(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	qs = qs.Limit(2).OrderBy("-id")

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" ORDER BY "simpleModel1"."id" DESC LIMIT 2`
	expectedArgs := []interface{}{}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 3, ret[0].ID)
	require.Equal(t, "Test3", ret[0].Title)
	require.Equal(t, "Test3", ret[0].Description)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "Test2", ret[1].Title)
	require.Equal(t, "Test2", ret[1].Description)
}

func TestAllLimitOffset(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	qs = qs.Limit(2).Offset(1)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" LIMIT 2 OFFSET 1`
	expectedArgs := []interface{}{}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, "Test2", ret[0].Title)
	require.Equal(t, "Test2", ret[0].Description)
	require.Equal(t, 3, ret[1].ID)
	require.Equal(t, "Test3", ret[1].Title)
	require.Equal(t, "Test3", ret[1].Description)
}

func TestGetOrNilGeneric(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel1](psql)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1) LIMIT 1`
	expectedArgs := []interface{}{1}
	actualQuery, actualArgs := qs.GetOrNilQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.GetOrNil()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 1, ret.ID)
	require.Equal(t, "Test", ret.Title)
	require.Equal(t, "Test", ret.Description)
}

func TestAllDefaultOrderBy(t *testing.T) {
	qs := newPsqlQuery[simpleModel2](t)
	createTestEntries2(t, qs.(*basePsql[simpleModel2]).psql)

	expectedQuery := `SELECT "simpleModel2"."id", "simpleModel2"."created_at", "simpleModel2"."name" FROM "simple_model_2" "simpleModel2" ORDER BY "simpleModel2"."created_at" DESC`
	expectedArgs := []interface{}{}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 3, len(ret))
	require.Equal(t, 3, ret[0].ID)
	require.Equal(t, "NewVersion", ret[0].Name)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "NewDistro", ret[1].Name)
	require.Equal(t, 1, ret[2].ID)
	require.Equal(t, "NewEmployee", ret[2].Name)
}

func TestAllMixedAndOrCombined(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	args.Set("title", "Test")
	args.Set("id2", 2)
	args.Set("title2", "Test2")
	qs = qs.
		Filter("id=:id", "title=:title").
		FilterOr("id=:id2", "title=:title2").
		Args(args).
		OrderBy("id")

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1 AND "simpleModel1"."title" = $2) OR ("simpleModel1"."id" = $3 AND "simpleModel1"."title" = $4) ORDER BY "simpleModel1"."id" ASC`
	expectedArgs := []interface{}{1, "Test", 2, "Test2"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, "Test", ret[0].Title)
	require.Equal(t, "Test", ret[0].Description)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "Test2", ret[1].Title)
	require.Equal(t, "Test2", ret[1].Description)
}

func TestAllMixedAndOrInOr(t *testing.T) {
	qs := newPsqlQuery[simpleModel1](t)
	createTestEntries(t, qs.(*basePsql[simpleModel1]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	args.Set("id2", 2)
	args.Set("title", "Test")
	args.Set("title2", "Test2")
	qs = qs.
		FilterInnerOr("id=:id", "id=:id2").
		FilterOrInnerOr("title=:title", "title=:title2").
		Args(args).
		OrderBy("id")

	expectedQuery := `SELECT "simpleModel1"."id", "simpleModel1"."title", "simpleModel1"."description" FROM "simple_model_1" "simpleModel1" WHERE ("simpleModel1"."id" = $1 OR "simpleModel1"."id" = $2) OR ("simpleModel1"."title" = $3 OR "simpleModel1"."title" = $4) ORDER BY "simpleModel1"."id" ASC`
	expectedArgs := []interface{}{1, 2, "Test", "Test2"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, "Test", ret[0].Title)
	require.Equal(t, "Test", ret[0].Description)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, "Test2", ret[1].Title)
	require.Equal(t, "Test2", ret[1].Description)
}

func TestAllOperators(t *testing.T) {
	// Filter keys can also contain various hints:
	// 	- "__ne" to negate the filter
	// 	- "__in" to use an IN clause
	// 	- "__nin" to use a NOT IN clause
	// 	- "__gt" to use a > clause
	// 	- "__gte" to use a >= clause
	// 	- "__lt" to use a < clause
	// 	- "__lte" to use a <= clause
	// 	- "__like" to use a LIKE clause
	// 	- "__nlike" to use a NOT LIKE clause
	// 	- "__ilike" to use a ILIKE clause
	// 	- "__nilike" to use a NOT ILIKE clause
	//  - "__null" to use a IS NULL clause
	//  - "__notnull" to use a IS NOT NULL clause
	//  - "__or" to prepend with OR instead of AND (in AND filter calls)
	//  - "__and" to prepend with AND instead of OR (in OR filter calls)
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	// First do negation
	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	qs = qs.Filter("id__ne=:id").Args(args)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."id" != $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{1}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)

	// Now do IN
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("id", pq.Int32Array{1, 2})
	qs = qs.Filter("id__in=:id").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."id" = ANY($1)) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{pq.Int32Array{1, 2}}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, 2, ret[1].ID)

	// Now do NOT IN
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("id", pq.Int32Array{1, 2})
	qs = qs.Filter("id__nin=:id").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."id" != ALL($1)) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{pq.Int32Array{1, 2}}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)
	require.NotNil(t, ret)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 3, ret[0].ID)

	// Now do >
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("num", 2)
	qs = qs.Filter("num__gt=:num").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."num" > $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{2}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)

	// Now do >=
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("num", 99999)
	qs = qs.Filter("num__gte=:num").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."num" >= $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{99999}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 3, ret[0].ID)

	// Now do <
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("num", 2)
	qs = qs.Filter("num__lt=:num").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."num" < $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{2}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)

	// Now do <=
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("num", 1337)
	qs = qs.Filter("num__lte=:num").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."num" <= $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{1337}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, 2, ret[1].ID)

	// Now do LIKE
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "MATCH THIS")
	qs = qs.Filter("non_nullable__like=%:non_nullable%").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" LIKE '%' || $1 || '%') ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"MATCH THIS"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)

	// Now do LIKE with wildcard in args
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "%MATCH THIS%")
	qs = qs.Filter("non_nullable__like=:non_nullable").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" LIKE $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"%MATCH THIS%"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)

	// Now do NOT LIKE
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "MATCH THIS")
	qs = qs.Filter("non_nullable__nlike=%:non_nullable%").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" NOT LIKE '%' || $1 || '%') ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"MATCH THIS"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)

	// Now do NOT LIKE with wildcard in args
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "%MATCH THIS%")
	qs = qs.Filter("non_nullable__nlike=:non_nullable").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" NOT LIKE $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"%MATCH THIS%"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)

	// Now do ILIKE
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "match this")
	qs = qs.Filter("non_nullable__ilike=%:non_nullable%").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" ILIKE '%' || $1 || '%') ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"match this"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)

	// Now do ILIKE with wildcard in args
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "%match this%")
	qs = qs.Filter("non_nullable__ilike=:non_nullable").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" ILIKE $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"%match this%"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)

	// Now do NOT ILIKE
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "match this")
	qs = qs.Filter("non_nullable__nilike=%:non_nullable%").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" NOT ILIKE '%' || $1 || '%') ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"match this"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)

	// Now do NOT ILIKE with wildcard in args
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("non_nullable", "%match this%")
	qs = qs.Filter("non_nullable__nilike=:non_nullable").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."non_nullable" NOT ILIKE $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{"%match this%"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)

	// Now do IS NULL
	qs = Q[simpleModel3](psql)
	qs = qs.Filter("nullable__null=true")

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."nullable" IS NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)

	// Now do IS NOT NULL
	qs = Q[simpleModel3](psql)
	qs = qs.Filter("nullable__notnull=true")

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."nullable" IS NOT NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 1, len(ret))
	require.Equal(t, 2, ret[0].ID)

	// Now do OR
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	qs = qs.Filter("nullable__notnull=true", "id__or=:id").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."nullable" IS NOT NULL OR "simpleModel3"."id" = $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{1}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, 2, ret[1].ID)

	// Now do AND
	qs = Q[simpleModel3](psql)
	args = orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	qs = qs.FilterInnerOr("nullable__notnull=true", "id__and=:id").Args(args)

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."nullable" IS NOT NULL AND "simpleModel3"."id" = $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{1}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 0, len(ret))

	// Now do OR with multiple hints
	qs = Q[simpleModel3](psql)
	qs = qs.Filter("nullable__notnull=true", "nullable__null__or=true")

	expectedQuery = `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."nullable" IS NOT NULL OR "simpleModel3"."nullable" IS NULL) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs = []interface{}{}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.All()
	require.Nil(t, err)

	require.Equal(t, 3, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, 2, ret[1].ID)
	require.Equal(t, 3, ret[2].ID)
}

func TestAllIsNotNullOr(t *testing.T) {
	psql := newPsql(t)
	qs := Q[simpleModel3](psql)
	createTestEntries3(t, qs.(*basePsql[simpleModel3]).psql)

	args := orderedmap.New[string, interface{}]()
	args.Set("id", 1)
	qs = qs.FilterInnerOr("nullable__notnull=true", "id=:id").Args(args)

	expectedQuery := `SELECT "simpleModel3"."id", "simpleModel3"."num", "simpleModel3"."non_nullable", "simpleModel3"."nullable" FROM "simple_model_3" "simpleModel3" WHERE ("simpleModel3"."nullable" IS NOT NULL OR "simpleModel3"."id" = $1) ORDER BY "simpleModel3"."id" ASC`
	expectedArgs := []interface{}{1}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)

	require.Equal(t, 2, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, 2, ret[1].ID)
}

func TestCreate(t *testing.T) {
	psql := newPsql(t)
	createTestModelCreate(t, psql)
	qs := Q[simpleModelCreate](psql)

	// Create a new entry
	entry := simpleModelCreate{
		Title:       "test",
		Description: "test-description",
	}

	expectedQuery := `INSERT INTO "simple_model_create" ("title", "description") VALUES ($1, $2) RETURNING "id", "title", "description"`
	expectedArgs := []interface{}{"test", "test-description"}
	actualQuery, actualArgs := qs.CreateQuery(&entry)
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	err := qs.Create(&entry)
	require.Nil(t, err)
	require.Equal(t, 1, entry.ID)

	// Select the entry and check if it is the same
	qs = Q[simpleModelCreate](psql)
	args := NewArgs()
	args.Set("id", entry.ID)
	args.Set("title", entry.Title)
	args.Set("description", entry.Description)
	qs = qs.Filter("id=:id", "title=:title", "description=:description").Args(args)

	x, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 1, len(x))
	require.Equal(t, entry.ID, x[0].ID)
	require.Equal(t, entry.Title, x[0].Title)
	require.Equal(t, entry.Description, x[0].Description)
}

func TestUpdate(t *testing.T) {
	psql := newPsql(t)
	createTestModelCreate(t, psql)
	qs := Q[simpleModelCreate](psql)

	// Create a new entry
	entry := simpleModelCreate{
		Title:       "test",
		Description: "test-description",
	}

	err := qs.Create(&entry)
	require.Nil(t, err)
	require.Equal(t, 1, entry.ID)

	// Update the entry
	qs = Q[simpleModelCreate](psql)
	args := NewArgs()
	args.Set("id", entry.ID)
	qs = qs.Filter("id=:id").Args(args)
	entry.Title = "test2"
	entry.Description = "test-description2"

	expectedQuery := `UPDATE "simple_model_create" SET "title" = $2, "description" = $3 WHERE ("id" = $1) RETURNING "id", "title", "description"`
	expectedArgs := []interface{}{1, "test2", "test-description2"}
	actualQuery, actualArgs := qs.UpdateQuery(&entry)
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	err = qs.Update(&entry)
	require.Nil(t, err)

	// Select the entry and check if it is the same
	qs = Q[simpleModelCreate](psql)
	args = NewArgs()
	args.Set("id", entry.ID)
	args.Set("title", entry.Title)
	args.Set("description", entry.Description)
	qs = qs.Filter("id=:id", "title=:title", "description=:description").Args(args)

	x, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 1, len(x))
	require.Equal(t, entry.ID, x[0].ID)
	require.Equal(t, entry.Title, x[0].Title)
	require.Equal(t, entry.Description, x[0].Description)
}

func TestTransactionUpdate(t *testing.T) {
	psql := newPsql(t)
	createTestModelCreate(t, psql)
	qs := Q[simpleModelCreate](psql)

	// Create a new entry
	entry := simpleModelCreate{
		Title:       "test",
		Description: "test-description",
	}

	err := qs.Create(&entry)
	require.Nil(t, err)
	require.Equal(t, 1, entry.ID)

	// start a transaction
	err = psql.Begin(context.TODO())
	defer psql.Rollback()
	require.Nil(t, err)

	// Update the entry
	qs = Q[simpleModelCreate](psql)
	args := NewArgs()
	args.Set("id", entry.ID)
	qs = qs.Filter("id=:id").Args(args)
	entry.Title = "test2"
	entry.Description = "test-description2"
	err = qs.Update(&entry)
	require.Nil(t, err)

	// commit the transaction
	err = psql.Commit()
	require.Nil(t, err)

	// Select the entry and check if it is the same
	qs = Q[simpleModelCreate](psql)
	args = NewArgs()
	args.Set("id", entry.ID)
	args.Set("title", entry.Title)
	args.Set("description", entry.Description)
	qs = qs.Filter("id=:id", "title=:title", "description=:description").Args(args)

	x, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 1, len(x))
	require.Equal(t, entry.ID, x[0].ID)
	require.Equal(t, entry.Title, x[0].Title)
	require.Equal(t, entry.Description, x[0].Description)
}

func TestTransactionRollback(t *testing.T) {
	psql := newPsql(t)
	createTestModelCreate(t, psql)
	qs := Q[simpleModelCreate](psql)

	// Create a new entry
	originalTitle := "test"
	originalDescription := "test-description"
	entry := simpleModelCreate{
		Title:       originalTitle,
		Description: originalDescription,
	}

	err := qs.Create(&entry)
	require.Nil(t, err)
	require.Equal(t, 1, entry.ID)

	// start a transaction
	err = psql.Begin(context.TODO())
	defer psql.Rollback()
	require.Nil(t, err)

	// Update the entry
	qs = Q[simpleModelCreate](psql)
	args := NewArgs()
	args.Set("id", entry.ID)
	qs = qs.Filter("id=:id").Args(args)
	entry.Title = "test2"
	entry.Description = "test-description2"
	err = qs.Update(&entry)
	require.Nil(t, err)

	// commit the transaction
	err = psql.Rollback()
	require.Nil(t, err)

	// try to find the updated entry
	qs = Q[simpleModelCreate](psql)
	args = NewArgs()
	args.Set("id", entry.ID)
	args.Set("title", entry.Title)
	args.Set("description", entry.Description)
	qs = qs.Filter("id=:id", "title=:title", "description=:description").Args(args)

	x, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 0, len(x))

	// find the original entry
	qs = Q[simpleModelCreate](psql)
	args = NewArgs()
	args.Set("id", entry.ID)
	qs = qs.Filter("id=:id").Args(args)

	x, err = qs.All()
	require.Nil(t, err)
	require.Equal(t, 1, len(x))
	require.Equal(t, entry.ID, x[0].ID)
	require.Equal(t, originalTitle, x[0].Title)
	require.Equal(t, originalDescription, x[0].Description)
}

func TestDelete(t *testing.T) {
	psql := newPsql(t)
	createTestModelCreate(t, psql)
	qs := Q[simpleModelCreate](psql)

	// Create a new entry
	entry := simpleModelCreate{
		Title:       "test",
		Description: "test-description",
	}

	err := qs.Create(&entry)
	require.Nil(t, err)
	require.Equal(t, 1, entry.ID)

	// Delete the entry
	qs = Q[simpleModelCreate](psql)
	args := NewArgs()
	args.Set("id", entry.ID)
	qs = qs.Filter("id=:id").Args(args)

	expectedQuery := `DELETE FROM "simple_model_create" WHERE ("id" = $1)`
	expectedArgs := []interface{}{1}
	actualQuery, actualArgs := qs.DeleteQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	err = qs.Delete()
	require.Nil(t, err)

	// Select the entry and check if it is the same
	qs = Q[simpleModelCreate](psql)
	args = NewArgs()
	args.Set("id", entry.ID)
	qs = qs.Filter("id=:id").Args(args)

	x, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 0, len(x))
}

func TestInOperatorWithArrayColumn(t *testing.T) {
	psql := newPsql(t)
	createTestModelWithArray(t, psql)

	// Search for an entry with an array column
	qs := Q[simpleModelWithArray](psql)
	args := NewArgs()
	args.Set("vars", "a")
	qs = qs.Filter("vars__in=:vars").Args(args)

	expectedQuery := `SELECT "simpleModelWithArray"."id", "simpleModelWithArray"."vars", "simpleModelWithArray"."varsint32" FROM "simple_model_with_array" "simpleModelWithArray" WHERE ($1 = ANY("simpleModelWithArray"."vars"))`
	expectedArgs := []interface{}{"a"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, pq.StringArray{"a", "b", "c"}, ret[0].Vars)
}

func TestNotInOperatorWithArrayColumn(t *testing.T) {
	psql := newPsql(t)
	createTestModelWithArray(t, psql)

	// Search for an entry with an array column
	qs := Q[simpleModelWithArray](psql)
	args := NewArgs()
	args.Set("vars", "a")
	qs = qs.Filter("vars__nin=:vars").Args(args)

	expectedQuery := `SELECT "simpleModelWithArray"."id", "simpleModelWithArray"."vars", "simpleModelWithArray"."varsint32" FROM "simple_model_with_array" "simpleModelWithArray" WHERE ($1 != ALL("simpleModelWithArray"."vars"))`
	expectedArgs := []interface{}{"a"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 2, len(ret))
	require.Equal(t, 2, ret[0].ID)
	require.Equal(t, 3, ret[1].ID)
	require.Equal(t, pq.StringArray{"d", "e", "f"}, ret[0].Vars)
	require.Equal(t, pq.StringArray{"g", "h", "i"}, ret[1].Vars)
}

func TestInOperatorWithArrayColumnInt32(t *testing.T) {
	psql := newPsql(t)
	createTestModelWithArray(t, psql)

	// Search for an entry with an array column
	qs := Q[simpleModelWithArray](psql)
	args := NewArgs()
	args.Set("varsint32", int32(1))
	qs = qs.Filter("varsint32__in=:varsint32").Args(args)

	expectedQuery := `SELECT "simpleModelWithArray"."id", "simpleModelWithArray"."vars", "simpleModelWithArray"."varsint32" FROM "simple_model_with_array" "simpleModelWithArray" WHERE ($1 = ANY("simpleModelWithArray"."varsint32"))`
	expectedArgs := []interface{}{int32(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.All()
	require.Nil(t, err)
	require.Equal(t, 1, len(ret))
	require.Equal(t, 1, ret[0].ID)
	require.Equal(t, pq.Int32Array{int32(1), int32(2), int32(3)}, ret[0].VarsInt32)
}

func TestSimpleJoinModel(t *testing.T) {
	psql := newPsql(t)
	createSimpleJoinModel(t, psql)

	qs := Q[joinSimpleModelMain](psql)
	args := NewArgs()
	args.Set("id", int32(1))
	qs = qs.InnerJoin(nil, joinModelForeign{}, "id", "foreign_key").Filter("id=:id").Args(args)

	expectedQuery := `SELECT "joinSimpleModelMain"."id", "joinSimpleModelMain"."name", "joinModelForeign"."id" FROM "join_simple_model_main" "joinSimpleModelMain" INNER JOIN "join_model_foreign" "joinModelForeign" ON "joinSimpleModelMain"."id" = "joinModelForeign"."foreign_key" WHERE ("joinSimpleModelMain"."id" = $1)`
	expectedArgs := []interface{}{int32(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)

	qs = qs.ClearAll().LeftJoin(nil, joinModelForeign{}, "id", "foreign_key").Filter("id=:id").Args(args)

	expectedQuery = `SELECT "joinSimpleModelMain"."id", "joinSimpleModelMain"."name", "joinModelForeign"."id" FROM "join_simple_model_main" "joinSimpleModelMain" LEFT JOIN "join_model_foreign" "joinModelForeign" ON "joinSimpleModelMain"."id" = "joinModelForeign"."foreign_key" WHERE ("joinSimpleModelMain"."id" = $1)`
	expectedArgs = []interface{}{int32(1)}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)

	qs = qs.ClearAll().RightJoin(nil, joinModelForeign{}, "id", "foreign_key").Filter("id=:id").Args(args)

	expectedQuery = `SELECT "joinSimpleModelMain"."id", "joinSimpleModelMain"."name", "joinModelForeign"."id" FROM "join_simple_model_main" "joinSimpleModelMain" RIGHT JOIN "join_model_foreign" "joinModelForeign" ON "joinSimpleModelMain"."id" = "joinModelForeign"."foreign_key" WHERE ("joinSimpleModelMain"."id" = $1)`
	expectedArgs = []interface{}{int32(1)}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)

	qs = qs.ClearAll().FullJoin(nil, joinModelForeign{}, "id", "foreign_key").Filter("id=:id").Args(args)

	expectedQuery = `SELECT "joinSimpleModelMain"."id", "joinSimpleModelMain"."name", "joinModelForeign"."id" FROM "join_simple_model_main" "joinSimpleModelMain" FULL JOIN "join_model_foreign" "joinModelForeign" ON "joinSimpleModelMain"."id" = "joinModelForeign"."foreign_key" WHERE ("joinSimpleModelMain"."id" = $1)`
	expectedArgs = []interface{}{int32(1)}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)
}

func TestComplexJoinModel(t *testing.T) {
	psql := newPsql(t)
	createComplexJoinModel(t, psql)

	qs := Q[joinComplexModelMain](psql)
	args := NewArgs()
	args.Set("id", int32(1))
	qs = qs.InnerJoin(nil, joinModelForeign{}, "id", "foreign_key").InnerJoin(joinModelForeign{}, joinModelAnotherForeign{}, "foreign_key", "id").Filter("id=:id").Args(args)

	expectedQuery := `SELECT "joinComplexModelMain"."id", "joinComplexModelMain"."name", "joinModelForeign"."id", "joinModelAnotherForeign"."id" FROM "join_complex_model_main" "joinComplexModelMain" INNER JOIN "join_model_foreign" "joinModelForeign" ON "joinComplexModelMain"."id" = "joinModelForeign"."foreign_key" INNER JOIN "join_model_another_foreign" "joinModelAnotherForeign" ON "joinModelForeign"."foreign_key" = "joinModelAnotherForeign"."id" WHERE ("joinComplexModelMain"."id" = $1)`
	expectedArgs := []interface{}{int32(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)
}

func TestSimpleSubquery(t *testing.T) {
	psql := newPsql(t)
	createSimpleJoinModel(t, psql)

	newqs := Q[joinSimpleModelMain](psql)
	newargs := NewArgs()
	newargs.Set("ivalue", "jason")
	newqs.Filter("name=:ivalue").Args(newargs).Include("id")

	qs := Q[joinModelForeign](psql)
	args := NewArgs()
	args.Set("value", int32(1))
	args.Set("query", newqs)
	qs = qs.Filter("id=:value").Filter("foreign_key__in=:query").Args(args)
	expectedQuery := `SELECT "joinModelForeign"."id", "joinModelForeign"."foreign_key" FROM "join_model_foreign" "joinModelForeign" WHERE ("joinModelForeign"."id" = $2) AND ("joinModelForeign"."foreign_key"  IN (SELECT "joinSimpleModelMain"."id" FROM "join_simple_model_main" "joinSimpleModelMain" WHERE ("joinSimpleModelMain"."name" = $1)))`
	expectedArgs := []interface{}{"jason", int32(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedArgs, actualArgs)
	require.Equal(t, expectedQuery, actualQuery)

	ret, err := qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)
}

func TestSimpleSubqueryWithValues(t *testing.T) {
	psql := newPsql(t)
	createSimpleJoinModel(t, psql)

	qs := Q[joinModelForeign](psql)
	args := NewArgs()
	args.Set("value", int32(1))
	args.Set("query", pq.Int32Array{1, 2, 3})
	qs = qs.Filter("id=:value").Filter("foreign_key__in=:query").Args(args)
	expectedQuery := `SELECT "joinModelForeign"."id", "joinModelForeign"."foreign_key" FROM "join_model_foreign" "joinModelForeign" WHERE ("joinModelForeign"."id" = $1) AND ("joinModelForeign"."foreign_key" = ANY($2))`
	expectedArgs := []interface{}{int32(1), pq.Int32Array{1, 2, 3}}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)
}

func TestComplexSubQuery(t *testing.T) {
	psql := newPsql(t)
	createComplexJoinModel(t, psql)

	foreignQS := Q[joinModelForeign](psql)
	foreignArgs := NewArgs()
	foreignArgs.Set("fid", int32(1))
	foreignQS.Filter("id=:fid").Args(foreignArgs).Include("foreign_key")

	anotherForeignQS := Q[joinModelAnotherForeign](psql)
	anotherForeignArgs := NewArgs()
	anotherForeignArgs.Set("product", "product")
	anotherForeignQS.Filter("product=:product").Args(anotherForeignArgs).Include("foreign_key")

	// InnerOr case
	qs := Q[joinComplexModelMain](psql)
	args := NewArgs()
	args.Set("fqs", foreignQS)
	args.Set("afqs", anotherForeignQS)
	qs = qs.FilterInnerOr("id__in=:fqs", "id__in=:afqs").Args(args)
	expectedQuery := `SELECT "joinComplexModelMain"."id", "joinComplexModelMain"."name", "joinModelForeign"."id", "joinModelAnotherForeign"."id" FROM "join_complex_model_main" "joinComplexModelMain","join_model_foreign" "joinModelForeign","join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinComplexModelMain"."id"  IN (SELECT "joinModelForeign"."foreign_key" FROM "join_model_foreign" "joinModelForeign" WHERE ("joinModelForeign"."id" = $1)) OR "joinComplexModelMain"."id"  IN (SELECT "joinModelAnotherForeign"."foreign_key" FROM "join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinModelAnotherForeign"."product" = $2)))`
	expectedArgs := []interface{}{int32(1), "product"}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)

	// InnerAnd case
	args = NewArgs()
	args.Set("fqs", foreignQS)
	args.Set("afqs", anotherForeignQS)
	qs = qs.ClearAll().Filter("id__in=:fqs", "id__in=:afqs").Args(args)
	expectedQuery = `SELECT "joinComplexModelMain"."id", "joinComplexModelMain"."name", "joinModelForeign"."id", "joinModelAnotherForeign"."id" FROM "join_complex_model_main" "joinComplexModelMain","join_model_foreign" "joinModelForeign","join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinComplexModelMain"."id"  IN (SELECT "joinModelForeign"."foreign_key" FROM "join_model_foreign" "joinModelForeign" WHERE ("joinModelForeign"."id" = $1)) AND "joinComplexModelMain"."id"  IN (SELECT "joinModelAnotherForeign"."foreign_key" FROM "join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinModelAnotherForeign"."product" = $2)))`
	expectedArgs = []interface{}{int32(1), "product"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)

	// OR case
	args = NewArgs()
	args.Set("fqs", foreignQS)
	args.Set("afqs", anotherForeignQS)
	qs = qs.ClearAll().FilterOr("id__in=:fqs").FilterOr("id__in=:afqs").Args(args)
	expectedQuery = `SELECT "joinComplexModelMain"."id", "joinComplexModelMain"."name", "joinModelForeign"."id", "joinModelAnotherForeign"."id" FROM "join_complex_model_main" "joinComplexModelMain","join_model_foreign" "joinModelForeign","join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinComplexModelMain"."id"  IN (SELECT "joinModelForeign"."foreign_key" FROM "join_model_foreign" "joinModelForeign" WHERE ("joinModelForeign"."id" = $1))) OR ("joinComplexModelMain"."id"  IN (SELECT "joinModelAnotherForeign"."foreign_key" FROM "join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinModelAnotherForeign"."product" = $2)))`
	expectedArgs = []interface{}{int32(1), "product"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)

	// AND case
	args = NewArgs()
	args.Set("fqs", foreignQS)
	args.Set("afqs", anotherForeignQS)
	qs = qs.ClearAll().Filter("id__in=:fqs").Filter("id__in=:afqs").Args(args)
	expectedQuery = `SELECT "joinComplexModelMain"."id", "joinComplexModelMain"."name", "joinModelForeign"."id", "joinModelAnotherForeign"."id" FROM "join_complex_model_main" "joinComplexModelMain","join_model_foreign" "joinModelForeign","join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinComplexModelMain"."id"  IN (SELECT "joinModelForeign"."foreign_key" FROM "join_model_foreign" "joinModelForeign" WHERE ("joinModelForeign"."id" = $1))) AND ("joinComplexModelMain"."id"  IN (SELECT "joinModelAnotherForeign"."foreign_key" FROM "join_model_another_foreign" "joinModelAnotherForeign" WHERE ("joinModelAnotherForeign"."product" = $2)))`
	expectedArgs = []interface{}{int32(1), "product"}
	actualQuery, actualArgs = qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err = qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)
}

func TestJoinFilter1(t *testing.T) {
	psql := newPsql(t)
	createSimpleJoinModel(t, psql)

	qs := Q[joinSimpleModelMain](psql)
	args := NewArgs()
	args.Set("id", int32(1))
	qs = qs.InnerJoin(nil, joinModelForeign{}, "id", "foreign_key").Filter("joinModelForeign.id=:id").Args(args)

	expectedQuery := `SELECT "joinSimpleModelMain"."id", "joinSimpleModelMain"."name", "joinModelForeign"."id" FROM "join_simple_model_main" "joinSimpleModelMain" INNER JOIN "join_model_foreign" "joinModelForeign" ON "joinSimpleModelMain"."id" = "joinModelForeign"."foreign_key" WHERE ("joinModelForeign"."id" = $1)`
	expectedArgs := []interface{}{int32(1)}
	actualQuery, actualArgs := qs.AllQuery()
	require.Equal(t, expectedQuery, actualQuery)
	require.Equal(t, expectedArgs, actualArgs)

	ret, err := qs.Get()
	require.Equal(t, err, nil)
	require.NotNil(t, ret)
	require.Equal(t, ret.ID, 1)
}
