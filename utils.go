// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.ciq.dev/pika/parser"
)

var (
	logger                *logrus.Logger
	pikaMetadataModelName = "PikaMetadataModelName"

	// Operators
	// Equal
	OpEq = "="
	// Not equal
	OpNeq = "!="
	// Greater than
	OpGt = ">"
	// Greater than or equal
	OpGte = ">="
	// Less than
	OpLt = "<"
	// Less than or equal
	OpLte = "<="
	// Like
	OpLike = "LIKE"
	// Not like
	OpNotLike = "NOT LIKE"
	// ILike
	OpILike = "ILIKE"
	// Not ILike
	OpNotILike = "NOT ILIKE"
	// Is null
	OpIsNull = "IS NULL"
	// Is not null
	OpIsNotNull = "IS NOT NULL"
	// Empty
	OpEmpty = ""

	// Hints
	// Negate
	HintNegate = "__ne"
	// In
	HintIn = "__in"
	// Not in
	HintNotIn = "__nin"
	// Greater than
	HintGt = "__gt"
	// Greater than or equal
	HintGte = "__gte"
	// Less than
	HintLt = "__lt"
	// Less than or equal
	HintLte = "__lte"
	// Like
	HintLike = "__like"
	// Not like
	HintNotLike = "__nlike"
	// ILike
	HintILike = "__ilike"
	// Not ILike
	HintNotILike = "__nilike"
	// Is null
	HintIsNull = "__null"
	// Is not null
	HintIsNotNull = "__notnull"
	// Or
	HintOr = "__or"
	// And
	HintAnd = "__and"
	// Empty
	HintEmpty = ""

	// Join Type
	innerJoin = "INNER JOIN"
	leftJoin  = "LEFT JOIN"
	rightJoin = "RIGHT JOIN"
	fullJoin  = "FULL JOIN"

	// Operators map
	operators = map[string]string{
		HintNegate:    OpNeq,
		HintIn:        OpEq,
		HintNotIn:     OpNeq,
		HintGt:        OpGt,
		HintGte:       OpGte,
		HintLt:        OpLt,
		HintLte:       OpLte,
		HintLike:      OpLike,
		HintNotLike:   OpNotLike,
		HintILike:     OpILike,
		HintNotILike:  OpNotILike,
		HintIsNull:    OpIsNull,
		HintIsNotNull: OpIsNotNull,
		HintEmpty:     OpEmpty,
	}

	// Antlr operator mapping
	antlrOperators = map[int]string{
		parser.FilterLexerEQUALS:         "__eq",
		parser.FilterLexerNOT_EQUALS:     HintNegate,
		parser.FilterLexerLESS_THAN:      HintLt,
		parser.FilterLexerLESS_EQUALS:    HintLte,
		parser.FilterLexerGREATER_EQUALS: HintGte,
		parser.FilterLexerGREATER_THAN:   HintGt,
		parser.FilterLexerCOLON:          HintILike,
	}
	antlrOperatorsNot = map[int]string{
		parser.FilterLexerEQUALS:         HintNegate,
		parser.FilterLexerNOT_EQUALS:     "__eq",
		parser.FilterLexerLESS_THAN:      HintGt,
		parser.FilterLexerLESS_EQUALS:    HintGte,
		parser.FilterLexerGREATER_EQUALS: HintLte,
		parser.FilterLexerGREATER_THAN:   HintLt,
		parser.FilterLexerCOLON:          HintNotILike,
	}

	// Antlr values
	antlrValues = map[int]bool{
		parser.FilterLexerSTRING:    true,
		parser.FilterLexerDURATION:  true,
		parser.FilterLexerTIMESTAMP: true,
		parser.FilterLexerNUM_FLOAT: true,
		parser.FilterLexerNUM_INT:   true,
		parser.FilterLexerNUM_UINT:  true,
		parser.FilterLexerTRUE:      true,
		parser.FilterLexerFALSE:     true,
		parser.FilterLexerNULL:      true,
	}
)

//nolint:gochecknoinits
func init() {
	logger = logrus.New()

	if os.Getenv("PIKA_DEBUG") == "1" {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
}

type pikaAip160State struct {
	initParens       bool
	activeParens     bool
	innerParens      bool
	activeOr         bool
	activeExpr       *pikaFiltering
	activeIdentifier string
	activeValue      any
	activeValueType  int
	activeOperator   string
	activeNot        bool
	forceNot         bool
	filterContent    []*pikaFiltering
	args             *orderedmap.OrderedMap[string, interface{}]
}

type pikaFiltering struct {
	entries *orderedmap.OrderedMap[string, string]
	or      bool
	innerOr bool
}

//nolint:structcheck
type base struct {
	filters        []pikaFiltering
	args           *orderedmap.OrderedMap[string, interface{}]
	excludeColumns []string
	includeColumns []string
	orderBy        []string
	ignoreLimit    bool
	ignoreOffset   bool
	ignoreOrderBy  bool
	limit          *int
	offset         *int
	err            error
	metadata       map[string]string
	joins          []*pikaJoin
	replaceFields  map[string]*replaceField
}

type pikaJoin struct {
	joinType string
	first    *joinInfo
	second   *joinInfo
}

// Join details
type joinInfo struct {
	tableName, modelName, key string
}

type replaceField struct {
	tableName string
	modelName string
}

type connBase struct {
	tableAlias map[string]string
}

func newBase() *base {
	return &base{
		args:          orderedmap.New[string, interface{}](),
		replaceFields: map[string]*replaceField{},
		joins:         make([]*pikaJoin, 0),
	}
}

func findEmptyForKey[T any](key string, f *orderedmap.OrderedMap[string, T]) string {
	if _, ok := f.Get(key); ok {
		return findEmptyForKey("!"+key, f)
	}

	return key
}

func cleanKey(key string) string {
	return strings.ReplaceAll(key, "!", "")
}

func (b *base) filter(innerOr bool, or bool, queries ...string) {
	newFilters := orderedmap.New[string, string]()
	for _, query := range queries {
		split := strings.Split(query, "=")
		if len(split) != 2 {
			b.err = fmt.Errorf("invalid filter: %s", query)
			break
		}
		if strings.Contains(split[0], "!") {
			b.err = fmt.Errorf("filter key contains exclamation mark: %s", query)
			break
		}
		newFilters.Set(findEmptyForKey(split[0], newFilters), split[1])
	}

	b.filters = append(b.filters, pikaFiltering{
		entries: newFilters,
		or:      or,
		innerOr: innerOr,
	})
}

func (b *base) setArgs(args *orderedmap.OrderedMap[string, interface{}]) {
	for pair := args.Oldest(); pair != nil; pair = pair.Next() {
		b.args.Set(pair.Key, pair.Value)
	}
}

func (b *base) clearAll() {
	b.args = orderedmap.New[string, interface{}]()
	b.filters = []pikaFiltering{}
	b.joins = []*pikaJoin{}
}

func (b *base) setLimit(limit int) {
	b.limit = &limit
}

func (b *base) setOffset(offset int) {
	b.offset = &offset
}

func (b *base) setOrderBy(orderBy []string, reset bool) {
	if reset {
		b.orderBy = []string{}
	}
	b.orderBy = append(b.orderBy, orderBy...)
}

func (c *connBase) TableAlias(src string, dst string) {
	if c.tableAlias == nil {
		c.tableAlias = make(map[string]string)
	}
	if _, ok := c.tableAlias[src]; ok {
		panic(fmt.Sprintf("duplicate table alias: %s", src))
	}
	c.tableAlias[src] = dst
}

func getPikaMetadata[T any]() map[string]string {
	x := struct {
		X T
	}{}

	metadata := make(map[string]string)

	ref := reflect.ValueOf(x.X)
	modelName := ref.Type().Name()
	metadata[pikaMetadataModelName] = modelName

	// Iterate through fields to get tags
	for i := 0; i < ref.NumField(); i++ {
		field := ref.Type().Field(i)

		// We only care about fields starting with "Pika"
		if strings.HasPrefix(field.Name, "Pika") {
			if contains(PikaMetadataFields, field.Name) {
				tag := field.Tag.Get("pika")

				if _, ok := metadata[field.Name]; ok {
					panic(fmt.Sprintf("duplicate Pika metadata field: %s", field.Name))
				}
				metadata[field.Name] = tag
			}
			continue
		}

		// This is a regular field, let's store information about it's type
		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}

		if _, ok := metadata[field.Name]; ok {
			panic(fmt.Sprintf("duplicate Pika database field: %s", field.Name))
		}

		metadata[tag] = field.Type.String()
	}

	return metadata
}

func contains[T comparable](slice []T, element T) bool {
	for _, e := range slice {
		if e == element {
			return true
		}
	}
	return false
}
