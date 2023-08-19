// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	// load psql driver
	_ "github.com/lib/pq"
)

// Queryable includes all methods shared by sqlx.DB and sqlx.Tx, allowing
// either type to be used interchangeably.
//
//nolint:interfacebloat
type Queryable interface {
	sqlx.Ext
	sqlx.ExecerContext
	sqlx.PreparerContext
	sqlx.QueryerContext
	sqlx.Preparer

	GetContext(context.Context, interface{}, string, ...interface{}) error
	SelectContext(context.Context, interface{}, string, ...interface{}) error
	Get(interface{}, string, ...interface{}) error
	MustExecContext(context.Context, string, ...interface{}) sql.Result
	PreparexContext(context.Context, string) (*sqlx.Stmt, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	Select(interface{}, string, ...interface{}) error
	QueryRow(string, ...interface{}) *sql.Row
	PrepareNamedContext(context.Context, string) (*sqlx.NamedStmt, error)
	PrepareNamed(string) (*sqlx.NamedStmt, error)
	Preparex(string) (*sqlx.Stmt, error)
	NamedExec(string, interface{}) (sql.Result, error)
	NamedExecContext(context.Context, string, interface{}) (sql.Result, error)
	MustExec(string, ...interface{}) sql.Result
	NamedQuery(string, interface{}) (*sqlx.Rows, error)
}

var (
	_ Queryable = (*sqlx.DB)(nil)
	_ Queryable = (*sqlx.Tx)(nil)
)

type PostgreSQL struct {
	*connBase
	db *sqlx.DB
	tx *sqlx.Tx
}

type basePsql[T any] struct {
	*AIPFilter[T]
	*PageToken[T]
	*base
	//nolint:structcheck // false positive
	psql *PostgreSQL
}

// NewPostgreSQL returns a new PostgreSQL instance.
// connectionString should be sqlx compatible.
func NewPostgreSQL(connectionString string) (*PostgreSQL, error) {
	db, err := sqlx.Connect("postgres", connectionString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	return &PostgreSQL{
		connBase: &connBase{},
		db:       db,
	}, nil
}

func NewPostgreSQLFromDB(db *sqlx.DB) *PostgreSQL {
	return &PostgreSQL{
		connBase: &connBase{},
		db:       db,
	}
}

// Begin starts a new transaction.
func (p *PostgreSQL) Begin(ctx context.Context) error {
	if p.tx != nil {
		return errors.New("transaction already exists")
	}

	tx, err := p.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	p.tx = tx

	return nil
}

// Commit commits the current transaction.
func (p *PostgreSQL) Commit() error {
	if p.tx != nil {
		defer func() {
			p.tx = nil
		}()

		return p.tx.Commit()
	}

	return errors.New("no transaction to commit")
}

// Rollback rolls back the current transaction.
func (p *PostgreSQL) Rollback() error {
	if p.tx != nil {
		defer func() {
			p.tx = nil
		}()

		return p.tx.Rollback()
	}

	return nil
}

func (p *PostgreSQL) Queryable() Queryable {
	if p.tx != nil {
		return p.tx
	}

	return p.db
}

func (p *PostgreSQL) DB() *sqlx.DB {
	return p.db
}

func (p *PostgreSQL) Close() error {
	return p.db.Close()
}

func PSQLQuery[T any](p *PostgreSQL) QuerySet[T] {
	b := &basePsql[T]{
		AIPFilter: NewAIPFilter[T](),
		PageToken: NewPageToken[T](),
		base:      newBase(),
		psql:      p,
	}

	// Initialize metadata once
	metadata := getPikaMetadata[T]()
	b.metadata = metadata

	modelName := b.metadata[pikaMetadataModelName]
	tableName := b.metadata[PikaMetadataTableName]

	// Check if we have a table alias for this model
	// Only applies if the table name is not explicitly set
	if tableName == "" {
		if x, ok := b.psql.tableAlias[modelName]; ok {
			// If so, use it
			tableName = x
		} else {
			// Otherwise, use the pluralized model name
			tableName = strcase.ToSnake(pluralize.NewClient().Plural(modelName))
		}
		b.metadata[PikaMetadataTableName] = tableName
	}

	return b
}

// Filter returns a new QuerySet with the given filters applied.
// The filters are applied in the order they are given.
// Only use named parameters in the filters.
// Multiple filter calls can be made, they will be combined with AND.
// Will also work as AND combined
func (b *basePsql[T]) Filter(queries ...string) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.filter(false, false, queries...)

	return b
}

// FilterOr returns a new QuerySet with the given filters applied.
// The filters are applied in the order they are given.
// Only use named parameters in the filters.
// Multiple filter calls can be made, they will be combined with AND.
// But will work as OR combined
func (b *basePsql[T]) FilterOr(queries ...string) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.filter(false, true, queries...)

	return b
}

// FilterInnerOr returns a new QuerySet with the given filters applied.
// Same as Filter, but inner filters are combined with OR.
func (b *basePsql[T]) FilterInnerOr(queries ...string) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.filter(true, false, queries...)

	return b
}

// FilterOrInnerOr returns a new QuerySet with the given filters applied.
// Same as FilterOr, but inner filters are combined with OR.
func (b *basePsql[T]) FilterOrInnerOr(queries ...string) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.filter(true, true, queries...)

	return b
}

// Args sets named arguments for the filters.
func (b *basePsql[T]) Args(args *orderedmap.OrderedMap[string, interface{}]) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.setArgs(args)

	return b
}

// ClearFiltersArgs clears the filters and args
func (b *basePsql[T]) ClearAll() QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.clearAll()

	return b
}

// Create creates a new record in the database.
func (b *basePsql[T]) Create(x *T) error {
	if b.err != nil {
		return b.err
	}

	origIgnoreOrderBy := b.ignoreOrderBy
	b.ignoreOrderBy = true
	q, args := b.CreateQuery(x)
	b.ignoreOrderBy = origIgnoreOrderBy

	// Execute query
	err := b.psql.Queryable().Get(x, q, args...)
	if err != nil {
		return err
	}

	return nil
}

// Update updates a record in the database.
func (b *basePsql[T]) Update(x *T) error {
	if b.err != nil {
		return b.err
	}

	origIgnoreOrderBy := b.ignoreOrderBy
	b.ignoreOrderBy = true
	q, args := b.UpdateQuery(x)
	b.ignoreOrderBy = origIgnoreOrderBy

	// Execute query
	err := b.psql.Queryable().Get(x, q, args...)
	if err != nil {
		return err
	}

	return nil
}

// Delete deletes a record from the database.
func (b *basePsql[T]) Delete() error {
	if b.err != nil {
		return b.err
	}

	origIgnoreOrderBy := b.ignoreOrderBy
	b.ignoreOrderBy = true
	q, args := b.DeleteQuery()
	b.ignoreOrderBy = origIgnoreOrderBy

	// Execute query
	_, err := b.psql.Queryable().Exec(q, args...)
	if err != nil {
		return err
	}

	return nil
}

// GetOrNil returns a single value or nil
// Multiple values will return an error.
func (b *basePsql[T]) GetOrNil() (*T, error) {
	if b.err != nil {
		return nil, b.err
	}

	q, args := b.GetOrNilQuery()

	// Execute query
	var x T

	// Send arguments to prepared statement
	err := b.psql.Queryable().Get(&x, q, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &x, nil
}

// Get returns a single value
// Returns error if no value is found
// Returns error if multiple values are found
func (b *basePsql[T]) Get() (*T, error) {
	if b.err != nil {
		return nil, b.err
	}

	q, args := b.GetQuery()

	// Execute query
	var x T

	// Send arguments to prepared statement
	err := b.psql.Queryable().Get(&x, q, args...)
	if err != nil {
		return nil, err
	}

	return &x, nil
}

// All returns all values
func (b *basePsql[T]) All() ([]*T, error) {
	if b.err != nil {
		return nil, b.err
	}

	q, args := b.AllQuery()

	// Execute query
	var x []*T

	// Send arguments to prepared statement
	err := b.psql.Queryable().Select(&x, q, args...)
	if err != nil {
		return nil, err
	}

	return x, nil
}

// Count returns the number of values
func (b *basePsql[T]) Count() (int, error) {
	if b.err != nil {
		return 0, b.err
	}

	// Execute query
	var x int

	// Fetch count without limit and offset
	origIgnoreLimit := b.ignoreLimit
	origIgnoreOffset := b.ignoreOffset
	origIgnoreOrderBy := b.ignoreOrderBy
	b.ignoreLimit = true
	b.ignoreOffset = true
	b.ignoreOrderBy = true
	filterStatement, args := b.queryWithFilters()
	preSelect := b.psqlSelectList(b.excludeColumns, b.includeColumns, false)
	// Strip preSelect from filterStatement
	filterStatement = strings.Replace(filterStatement, preSelect, "", -1)
	b.ignoreLimit = origIgnoreLimit
	b.ignoreOffset = origIgnoreOffset
	b.ignoreOrderBy = origIgnoreOrderBy

	// Get select query and append filter statement
	selectQuery := b.psqlCountQuery()
	q := fmt.Sprintf("%s%s", selectQuery, filterStatement)
	logger.Debugf("Pika query: %s", q)

	err := b.psql.Queryable().Get(&x, q, args...)
	if err != nil {
		return 0, err
	}

	return x, nil
}

// Limit sets the limit for the query
func (b *basePsql[T]) Limit(limit int) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.setLimit(limit)

	return b
}

// Offset sets the offset for the query
func (b *basePsql[T]) Offset(offset int) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.setOffset(offset)

	return b
}

// OrderBy sets the order for the query
// Use - to indicate descending order
// Example:
//
//	OrderBy("-id", "name")
func (b *basePsql[T]) OrderBy(order ...string) QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.setOrderBy(order, false)

	return b
}

// ResetOrderBy resets the order for the query
func (b *basePsql[T]) ResetOrderBy() QuerySet[T] {
	if b.err != nil {
		return b
	}

	b.setOrderBy([]string{}, true)

	return b
}

// CreateQuery returns the query and arguments for Create
func (b *basePsql[T]) CreateQuery(x *T) (string, []interface{}) {
	q, args := b.psqlCreateQuery(x)
	logger.Debugf("Pika query: %s", q)

	return q, args
}

// UpdateQuery returns the query and arguments for Update
func (b *basePsql[T]) UpdateQuery(x *T) (string, []interface{}) {
	q, args := b.psqlUpdateQuery(x)
	logger.Debugf("Pika query: %s", q)

	return q, args
}

// DeleteQuery returns the query and arguments for Delete
func (b *basePsql[T]) DeleteQuery() (string, []interface{}) {
	modelName := b.metadata[pikaMetadataModelName]

	filterStatement, args := b.filterStatement()
	filterStatement = strings.Replace(filterStatement, fmt.Sprintf("\"%s\".", modelName), "", -1)

	q := fmt.Sprintf("DELETE FROM \"%s\"", b.metadata[PikaMetadataTableName])
	q += filterStatement
	logger.Debugf("Pika query: %s", q)

	return q, args
}

func (b *basePsql[T]) GetOrNilQuery() (string, []interface{}) {
	b.ignoreLimit = true
	q, args := b.queryWithFilters()

	// Limit to one
	q += " LIMIT 1"

	logger.Debugf("Pika query: %s", q)

	return q, args
}

// GetQuery returns the query and arguments for Get
func (b *basePsql[T]) GetQuery() (string, []interface{}) {
	q, args := b.GetOrNilQuery()
	return q, args
}

// AllQuery returns the query and arguments for All
func (b *basePsql[T]) AllQuery() (string, []interface{}) {
	q, args := b.queryWithFilters()
	logger.Debugf("Pika query: %s", q)

	return q, args
}

// AIP160 filtering for gRPC/Proto
func (b *basePsql[T]) AIP160(filter string, options AIPFilterOptions) (QuerySet[T], error) {
	if b.err != nil {
		return b, b.err
	}

	return b.aip160(b, filter, options)
}

// Page tokens for gRPC
func (b *basePsql[T]) GetPage(paginatable Paginatable, options AIPFilterOptions) ([]*T, string, error) {
	if b.err != nil {
		return nil, "", b.err
	}

	// Only decode if token is not empty
	if paginatable.GetPageToken() != "" {
		err := b.PageToken.Decode(paginatable.GetPageToken())
		if err != nil {
			return nil, "", err
		}
	} else {
		// Otherwise use initial filter
		b.PageToken.Offset = 0
		b.PageToken.Filter = paginatable.GetFilter()
		b.PageToken.OrderBy = paginatable.GetOrderBy()
		b.PageToken.PageSize = uint(paginatable.GetPageSize())
	}

	qs, err := b.pageToken(b, options)
	if err != nil {
		return nil, "", err
	}

	result, err := qs.All()
	if err != nil {
		return nil, "", err
	}

	b.PageToken.Offset += uint(len(result))

	// Get count and check if there are more results
	count, err := b.Count()
	if err != nil {
		return nil, "", fmt.Errorf("getting count: %w", err)
	}

	// If no more results after this page, return empty page token
	if b.PageToken.Offset >= uint(count) {
		return result, "", nil
	}

	tk, err := b.PageToken.Encode()
	if err != nil {
		return nil, "", err
	}

	return result, tk, nil
}

func (b *basePsql[T]) InnerJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T] {
	return b.commonJoin(innerJoin, modelFirst, modelSecond, keyFirst, keySecond)
}

func (b *basePsql[T]) LeftJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T] {
	return b.commonJoin(leftJoin, modelFirst, modelSecond, keyFirst, keySecond)
}

func (b *basePsql[T]) RightJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T] {
	return b.commonJoin(rightJoin, modelFirst, modelSecond, keyFirst, keySecond)
}

func (b *basePsql[T]) FullJoin(modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T] {
	return b.commonJoin(fullJoin, modelFirst, modelSecond, keyFirst, keySecond)
}

// Exclude certain fields (Notice: should be fields defined in "db" or "pika")
func (b *basePsql[T]) Exclude(excludes ...string) QuerySet[T] {
	if b.err != nil {
		return b
	}
	if len(excludes) > 0 {
		if b.excludeColumns == nil {
			b.excludeColumns = make([]string, 0)
		}
		b.excludeColumns = append(b.excludeColumns, excludes...)
	}

	return b
}

// Include certain fields (Notice: should be fields defined in "db" or "pika")
func (b *basePsql[T]) Include(includes ...string) QuerySet[T] {
	if b.err != nil {
		return b
	}
	if len(includes) > 0 {
		if b.includeColumns == nil {
			b.includeColumns = make([]string, 0)
		}
		b.includeColumns = append(b.includeColumns, includes...)
	}

	return b
}

// Return args, used for reflection
func (b *basePsql[T]) GetArgs() *orderedmap.OrderedMap[string, interface{}] {
	return b.args
}

// Return current table and module name, used for reflection
func (b *basePsql[T]) GetModel() (string, string) {
	var x T
	modelName := reflect.TypeOf(x).Name()
	tableName := modelName
	ref := reflect.ValueOf(x)
	for i := 0; i < ref.NumField(); i++ {
		field := ref.Type().Field(i)
		if strings.Compare(field.Name, PikaMetadataTableName) == 0 {
			tableName = field.Tag.Get("pika")
			break
		}
	}

	return tableName, modelName
}

type subQuery struct {
	query string
	args  *orderedmap.OrderedMap[string, interface{}]
}

func (b *basePsql[T]) filterStatement() (string, []any) {
	q := ""

	// Set number args for named parameters
	mapping := make(map[string]int)
	reverseMapping := make(map[int]string)
	// Save subquery info
	subQueryMap := make(map[string]*subQuery)
	// Rearranged args, as subqueries have their own place holders, so we need to rearrange the entire place holders.
	newArgsMap := orderedmap.New[string, interface{}]()

	// Process filters if any
	if len(b.filters) > 0 {
		// Map args to numbers
		// And reverse mapping to easily get the name
		if b.args.Len() > 0 {
			start := 1
			// First scan subquery
			// Userd for rearrange subquery args
			for pair := b.args.Oldest(); pair != nil; pair = pair.Next() {
				k := pair.Key

				// already processed
				if _, ok := mapping[k]; ok {
					continue
				}

				v := pair.Value
				// Check if v is subquery, another QuerySet[T]
				if isTarget(v) {
					// Retrieve subquery type info
					tname, mname := getQuerySetInfo(v)
					b.replaceFields[tname] = &replaceField{
						tableName: tname,
						modelName: mname,
					}

					// Retrieve subquery details
					query, args := getSubQuery(v)
					if args.Len() > 0 {
						// update query
						oldIdx := *generateRangeSlice(1, args.Len())
						newIdx := *generateRangeSlice(start, args.Len())
						query = replacePlaceHolder(query, oldIdx, newIdx)

						for p := args.Oldest(); p != nil; p = p.Next() {
							mapping[p.Key] = start
							newArgsMap.AddPairs(*p)
							reverseMapping[start] = p.Key
							start++
						}
					}
					sq := subQuery{
						query: query,
						args:  args,
					}
					subQueryMap[k] = &sq
				}
			}

			// Update remaining args
			for pair := b.args.Oldest(); pair != nil; pair = pair.Next() {
				k := pair.Key

				// If already set, re-use same number
				if _, ok := mapping[k]; ok {
					continue
				}

				v := pair.Value
				if !isTarget(v) {
					newArgsMap.AddPairs(*pair)
					// Set number
					mapping[k] = start
					reverseMapping[start] = k
					start++
				}
			}
		}

		// Process filters
		q += " WHERE "
		for _, filter := range b.filters {
			// If no filters, then open with parenthesis
			innerQ := "("

			// Else
			// If not first filter, add AND/OR
			if !strings.HasSuffix(q, "WHERE ") {
				innerQ = " AND ("
				if filter.or {
					innerQ = " OR ("
				}
			}

			// Loop through filter entries
			for pair := filter.entries.Oldest(); pair != nil; pair = pair.Next() {
				// vSpace is used to determine if we need to add a space
				// Required only for IS NULL and IS NOT NULL
				vSpace := " "

				// Whether or not to switch left-hand side with right-hand side
				// This is used for IN and NOT IN where we're checking if a single value
				// is present in the array column value
				shouldSwitchKV := false

				// kWrapper is whether to wrap the key in a function
				// Mostly used for ANY and ALL when checking array columns
				keyWrapper := ""

				k := pair.Key
				v := pair.Value

				// If argument is set, use it
				// Only if the value starts with a ":"
				noWildcard := strings.ReplaceAll(v, "%", "")
				startWildcard := strings.HasPrefix(v, "%")
				endWildcard := strings.HasSuffix(v, "%")
				if strings.HasPrefix(noWildcard, ":") {
					// Allow a percentage sign to be used as a wildcard
					// Both prefix and suffix
					// Ignore it for the purposes of named parameters
					if _, ok := b.args.Get(noWildcard[1:]); ok {
						// If mapping found, replace with numbered parameter
						v = fmt.Sprintf("$%d", mapping[noWildcard[1:]])
					}
				}
				andOr := "AND"
				if filter.innerOr {
					andOr = "OR"
				}

				operator := "="
				// If key contains "__", then try to find hint
				if strings.Contains(k, "__") {
					parts := strings.Split(k, "__")
					k = parts[0]
					op := fmt.Sprintf("__%s", parts[1])

					// IN requires the value wrapped in ANY
					// as go-pika sends the value as a slice
					if op == HintIn {
						// If the field type is a StringArray, then switch left-hand side and right-hand side
						// This is because left-hand side cannot be ANY
						origV := v
						// If it's a variable pointing to subquery object
						if val, ok := subQueryMap[noWildcard[1:]]; ok {
							v = fmt.Sprintf("IN (%s)", val.query)
							// We do not need "="
							op = HintEmpty
						} else {
							v = fmt.Sprintf("ANY(%s)", v)
							if x, ok := b.metadata[k]; ok {
								if strings.HasPrefix(x, "pq.") && strings.HasSuffix(x, "Array") {
									v = origV
									shouldSwitchKV = true
									keyWrapper = "ANY"
								}
							}
						}
					}

					// NOT IN requires the value wrapped in ALL
					// as go-pika sends the value as a slice
					if op == HintNotIn {
						// If the field type is a StringArray, then switch left-hand side and right-hand side
						// This is because left-hand side cannot be ALL
						origV := v
						if val, ok := subQueryMap[noWildcard[1:]]; ok {
							v = fmt.Sprintf("NOT IN (%s)", val.query)
							op = HintEmpty
						} else {
							v = fmt.Sprintf("ALL(%s)", v)
							if x, ok := b.metadata[k]; ok {
								if strings.HasPrefix(x, "pq.") && strings.HasSuffix(x, "Array") {
									v = origV
									shouldSwitchKV = true
									keyWrapper = "ALL"
								}
							}
						}
					}

					// If LIKE or NOT LIKE, then respect wildcards
					// Also for not case sensitive variants
					if op == HintLike || op == HintNotLike || op == HintILike || op == HintNotILike {
						// If a start wildcard was found, then add a prefix
						if startWildcard {
							v = fmt.Sprintf("'%%' || %s", v)
						}

						// If an end wildcard was found, then add a suffix
						if endWildcard {
							v = fmt.Sprintf("%s || '%%'", v)
						}
					}

					// If IS NULL or IS NOT NULL, then ignore value
					if op == HintIsNull || op == HintIsNotNull {
						v = ""
						vSpace = ""
					}

					extraHintOp := op
					if len(parts) > 2 {
						extraHintOp = fmt.Sprintf("__%s", parts[2])
					}

					// If AND then set andOr to AND regardless of filter.innerOr
					// We do this by replacing last AND/OR with AND
					if op == HintAnd || extraHintOp == HintAnd {
						innerQ = strings.TrimSuffix(innerQ, "AND ")
						innerQ = strings.TrimSuffix(innerQ, "OR ")

						// Add if it's not start of subexpression
						if !strings.HasSuffix(innerQ, "(") {
							innerQ += "AND "
						}
					}

					// If OR then set andOr to OR regardless of filter.innerOr
					if op == HintOr || extraHintOp == HintOr {
						innerQ = strings.TrimSuffix(innerQ, "AND ")
						innerQ = strings.TrimSuffix(innerQ, "OR ")

						// Add if it's not start of subexpression
						if !strings.HasSuffix(innerQ, "(") {
							innerQ += "OR "
						}
					}

					// Check if operator is valid
					// Only if op is not HintAnd or HintOr
					if op != HintAnd && op != HintOr {
						var ok bool
						operator, ok = operators[op]
						if !ok {
							b.err = fmt.Errorf("invalid operator: %s", operator)
							return "", nil
						}
					}
				}

				clean := cleanKey(k)
				finalK := fmt.Sprintf("\"%s\".\"%s\"", b.metadata[pikaMetadataModelName], clean)
				// If there is a dot in cleanKey, then that means we should assume that
				// the caller "knows" what they're doing and we should not add the table name
				if strings.Contains(clean, ".") {
					// Split by dot, then join with quotes
					parts := strings.Split(clean, ".")
					if len(parts) != 2 {
						b.err = fmt.Errorf("invalid key: %s", k)
						return "", nil
					}
					finalK = fmt.Sprintf("\"%s\".\"%s\"", parts[0], parts[1])
				}
				if keyWrapper != "" {
					finalK = fmt.Sprintf("%s(%s)", keyWrapper, finalK)
				}

				if shouldSwitchKV {
					innerQ += fmt.Sprintf("%s %s %s%s%s ", v, operator, finalK, vSpace, andOr)
					continue
				}

				innerQ += fmt.Sprintf("%s %s %s%s%s ", finalK, operator, v, vSpace, andOr)
			}
			// Remove last AND and OR (and first)
			innerQ = strings.TrimSuffix(innerQ, " AND ")
			innerQ = strings.TrimSuffix(innerQ, " OR ")

			innerQ += ")"

			// Add to query
			q += innerQ
		}
	}

	// Process order by
	// If not ignored
	if !b.ignoreOrderBy {
		// Proceed if there are order bys
		if len(b.orderBy) > 0 {
			q += " ORDER BY "
			for _, o := range b.orderBy {
				if strings.HasPrefix(o, "-") {
					o = fmt.Sprintf("\"%s\".\"%s\" DESC", b.metadata[pikaMetadataModelName], o[1:])
				} else {
					o = fmt.Sprintf("\"%s\".\"%s\" ASC", b.metadata[pikaMetadataModelName], o)
				}
				q += o + ", "
			}
			// Remove last comma
			q = strings.TrimSuffix(q, ", ")
		} else if orderBy := b.metadata[PikaMetadataDefaultOrderBy]; orderBy != "" {
			q += " ORDER BY "
			if strings.HasPrefix(orderBy, "-") {
				orderBy = fmt.Sprintf("\"%s\".\"%s\" DESC", b.metadata[pikaMetadataModelName], orderBy[1:])
			} else {
				orderBy = fmt.Sprintf("\"%s\".\"%s\" ASC", b.metadata[pikaMetadataModelName], orderBy)
			}
			q += orderBy
		}
	}

	if b.limit != nil && !b.ignoreLimit {
		q += fmt.Sprintf(" LIMIT %d", *b.limit)
	}

	if b.offset != nil && !b.ignoreOffset {
		q += fmt.Sprintf(" OFFSET %d", *b.offset)
	}

	// Construct argument list
	// If we have subqueries, we return the newArgsMap instead of b.args, because args are already rearranged
	if newArgsMap.Len() > 0 {
		args := make([]interface{}, 0, newArgsMap.Len())
		for pair := newArgsMap.Oldest(); pair != nil; pair = pair.Next() {
			args = append(args, pair.Value)
		}
		logger.Debugf("Pika args: %v", args)
		return q, args
	}

	args := make([]interface{}, 0, b.args.Len())
	for pair := b.args.Oldest(); pair != nil; pair = pair.Next() {
		args = append(args, pair.Value)
	}
	logger.Debugf("Pika args: %v", args)
	return q, args
}

func (b *basePsql[T]) queryWithFilters() (string, []interface{}) {
	// Need to process filter first
	filterStatement, args := b.filterStatement()

	q := b.psqlSelectList(b.excludeColumns, b.includeColumns, false)
	// If we have joins, we need to modify the from str
	if len(b.joins) > 0 {
		queries := []string{q}
		for _, join := range b.joins {
			// It'll be the form of `join_type table2_name model2_name ON model1_name.key = model2_name.key`
			joinQ := fmt.Sprintf("%s \"%s\" \"%s\" ON \"%s\".\"%s\" = \"%s\".\"%s\"", join.joinType, join.second.tableName, join.second.modelName, join.first.modelName, join.first.key, join.second.modelName, join.second.key)
			queries = append(queries, joinQ)
		}
		q = strings.Join(queries, " ")
	}

	q += filterStatement

	return q, args
}

type column struct {
	db   string
	pika string
}

func (b *basePsql[T]) psqlSelectList(excludeColumns []string, includeColumns []string, onlyCols bool) string {
	// If nil, create empty slice
	if excludeColumns == nil {
		excludeColumns = make([]string, 0)
	}
	if includeColumns == nil {
		includeColumns = make([]string, 0)
	}

	// Get info from metadata
	tableName := b.metadata[PikaMetadataTableName]
	modelName := b.metadata[pikaMetadataModelName]

	// Create dummy instance of T
	var x T

	// Reflect value to fetch fields and tags
	ref := reflect.ValueOf(x)

	columns := make([]column, 0, ref.NumField())

	// Iterate through fields to get tags
	for i := 0; i < ref.NumField(); i++ {
		field := ref.Type().Field(i)
		tag := field.Tag.Get("db")
		// By default, pikaTag = tag
		pikaTag := tag
		if field.Tag.Get("pika") != "" {
			// If pikaTag is set, then it'll be that set value
			pikaTag = field.Tag.Get("pika")
		}

		// Ignore empty or "-" tags
		if tag == "" || tag == "-" {
			continue
		}

		// Check if we have a dedicated include list
		// and if the current tag is not in it
		// then skip it
		if len(includeColumns) > 0 && !contains(includeColumns, pikaTag) {
			continue
		}

		// Check if we have a dedicated exclude list
		// and if the current tag is in it
		// then skip it
		if len(excludeColumns) > 0 && contains(excludeColumns, pikaTag) {
			continue
		}

		columns = append(columns, column{
			db:   tag,
			pika: pikaTag,
		})
	}

	// Default from str
	fromStrs := []string{fmt.Sprintf("FROM \"%s\" \"%s\"", tableName, modelName)}

	// Prefix each column with the model name
	// to avoid conflicts
	var selectColumns []string
	for _, column := range columns {
		if column.pika != "" {
			values := strings.SplitN(column.pika, ".", 2)
			if len(values) == 2 {
				if val, ok := b.replaceFields[values[0]]; ok {
					// Need to replace fields from other tables with associated model prefixs
					// These fields are defined in the current model, but their values are from other tables
					selectColumns = append(selectColumns, fmt.Sprintf("\"%s\".\"%s\"", val.modelName, column.db))
					// If table and model names do NOT exist in joins, we need to add them to from str separately
					// Otherwise, models definitions are missing in the generated query
					if !b.checkJoins(val.tableName, val.modelName) {
						fromStrs = append(fromStrs, fmt.Sprintf("\"%s\" \"%s\"", val.tableName, val.modelName))
					}
					continue
				}
			}
		}
		selectColumns = append(selectColumns, fmt.Sprintf("\"%s\".\"%s\"", modelName, column.db))
	}

	if onlyCols {
		return strings.Join(selectColumns, ", ")
	}

	selectStr := fmt.Sprintf("SELECT %s", strings.Join(selectColumns, ", "))

	q := fmt.Sprintf("%s %s", selectStr, strings.Join(fromStrs, ","))
	return q
}

func (b *basePsql[T]) psqlCountQuery() string {
	// Table name, set it to empty first
	// but will be either set to snake cased
	// model name or the value of the "pika" tag
	// for the PikaTableName field.
	tableName := b.metadata[PikaMetadataTableName]
	modelName := b.metadata[pikaMetadataModelName]

	fromStr := fmt.Sprintf("FROM \"%s\" \"%s\"", tableName, modelName)

	selectStr := "SELECT COUNT(*)"

	q := fmt.Sprintf("%s %s", selectStr, fromStr)
	return q
}

func (b *basePsql[T]) psqlCreateQuery(value *T) (string, []any) {
	// Get info from metadata
	tableName := b.metadata[PikaMetadataTableName]
	modelName := b.metadata[pikaMetadataModelName]

	// Reflect value to fetch fields and tags
	ref := reflect.ValueOf(value)

	columns := make([]string, 0, ref.Elem().NumField())
	values := make([]string, 0, ref.Elem().NumField())

	// Iterate through fields to get tags
	xi := 0
	for i := 0; i < ref.Elem().NumField(); i++ {
		field := ref.Elem().Type().Field(i)
		tag := field.Tag.Get("db")
		// Ignore "-" tags (or empty tags)
		if tag == "" || tag == "-" {
			continue
		}

		// Ignore empty or "-" tags
		tagSplit := strings.Split(field.Tag.Get("pika"), ",")
		skipCol := false
		for _, t := range tagSplit {
			// If tag has "omitempty" and the value is empty
			// then skip it
			if t == "omitempty" {
				fieldValue := ref.Elem().Field(i)
				if reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(fieldValue.Type()).Interface()) {
					skipCol = true
					break
				}
			}
		}
		if skipCol {
			continue
		}

		colName := fmt.Sprintf("\"%s\"", tag)
		columns = append(columns, colName)
		values = append(values, fmt.Sprintf("$%d", xi+1))
		xi++
	}

	columnStr := strings.Join(columns, ", ")
	valueStr := strings.Join(values, ", ")

	selectList := b.psqlSelectList(b.excludeColumns, b.includeColumns, true)
	// Remove the model name prefix from the select list
	// since we are inserting into the table
	selectList = strings.Replace(selectList, fmt.Sprintf("\"%s\".", modelName), "", -1)
	q := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s) RETURNING %s", tableName, columnStr, valueStr, selectList)

	// Convert value to arguments
	args := make([]interface{}, 0, ref.Elem().NumField())
	for i := 0; i < ref.Elem().NumField(); i++ {
		field := ref.Elem().Type().Field(i)
		tag := fmt.Sprintf("\"%s\"", field.Tag.Get("db"))

		if !contains(columns, tag) {
			continue
		}

		fieldElem := ref.Elem().Field(i)
		args = append(args, fieldElem.Interface())
	}

	return q, args
}

func (b *basePsql[T]) psqlUpdateQuery(value *T) (string, []any) {
	// Get info from metadata
	tableName := b.metadata[PikaMetadataTableName]
	modelName := b.metadata[pikaMetadataModelName]

	// Reflect value to fetch fields and tags
	ref := reflect.ValueOf(value)

	columns := make([]string, 0, ref.Elem().NumField())
	// values := make([]string, 0, ref.Elem().NumField())

	// Iterate through fields to get tags
	xi := 0
	for i := 0; i < ref.Elem().NumField(); i++ {
		field := ref.Elem().Type().Field(i)
		tag := field.Tag.Get("db")
		// Ignore "-" tags (or empty tags)
		if tag == "" || tag == "-" {
			continue
		}

		// Skip ID field
		if tag == "id" {
			continue
		}

		// Ignore empty or "-" tags
		tagSplit := strings.Split(field.Tag.Get("pika"), ",")
		skipCol := false
		for _, t := range tagSplit {
			// If tag has "omitempty" and the value is empty
			// then skip it
			if t == "omitempty" {
				fieldValue := ref.Elem().Field(i)
				if reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(fieldValue.Type()).Interface()) {
					skipCol = true
					break
				}
			}
		}
		if skipCol {
			continue
		}

		colName := fmt.Sprintf("\"%s\"", tag)
		columns = append(columns, colName)
		// values = append(values, fmt.Sprintf("$%d", xi+1))
		xi++
	}

	filterStatement, args := b.filterStatement()
	if filterStatement == "" {
		b.err = errors.New("No filter statement found")
		return "", nil
	}

	selectList := b.psqlSelectList(b.excludeColumns, b.includeColumns, true)
	// Remove the model name prefix from the select list
	// since we are inserting into the table
	selectList = strings.Replace(selectList, fmt.Sprintf("\"%s\".", modelName), "", -1)
	q := fmt.Sprintf("UPDATE \"%s\" SET ", tableName)

	// Add columns to update
	for i, col := range columns {
		q += fmt.Sprintf("%s = $%d", col, i+1+b.args.Len())
		if i < len(columns)-1 {
			q += ", "
		}
	}

	// Add where clause
	filterStatement = strings.Replace(filterStatement, fmt.Sprintf("\"%s\".", modelName), "", -1)
	q += fmt.Sprintf("%s RETURNING %s", filterStatement, selectList)

	// Convert value to arguments
	for i := 0; i < ref.Elem().NumField(); i++ {
		field := ref.Elem().Type().Field(i)
		tag := fmt.Sprintf("\"%s\"", field.Tag.Get("db"))

		if !contains(columns, tag) {
			continue
		}

		fieldElem := ref.Elem().Field(i)
		args = append(args, fieldElem.Interface())
	}

	return q, args
}

// Check whether given table and module names are inside join array
func (b *basePsql[T]) checkJoins(tname, mname string) bool {
	for _, join := range b.joins {
		if (tname == join.first.tableName && mname == join.first.modelName) || (tname == join.second.tableName && mname == join.second.modelName) {
			return true
		}
	}

	return false
}

func (b *basePsql[T]) commonJoin(joinType string, modelFirst, modelSecond interface{}, keyFirst, keySecond string) QuerySet[T] {
	if b.err != nil {
		return b
	}

	if modelFirst == nil && modelSecond == nil {
		b.err = fmt.Errorf("modelFirst and modelSecond are all nil, this is not allowed")
		return b
	}

	var x T
	selftn, selfmn := getQuerySetInfo(x)

	var tnFirst, mnFirst, tnSecond, mnSecond string
	if modelFirst == nil {
		tnFirst, mnFirst = selftn, selfmn
	} else {
		tnFirst, mnFirst = getQuerySetInfo(modelFirst)
	}

	if modelSecond == nil {
		tnSecond, mnSecond = selftn, selfmn
	} else {
		tnSecond, mnSecond = getQuerySetInfo(modelSecond)
	}

	b.joins = append(b.joins, &pikaJoin{
		joinType: joinType,
		first: &joinInfo{
			tableName: tnFirst,
			modelName: mnFirst,
			key:       keyFirst,
		},
		second: &joinInfo{
			tableName: tnSecond,
			modelName: mnSecond,
			key:       keySecond,
		},
	})

	b.replaceFields[tnFirst] = &replaceField{
		tableName: tnFirst,
		modelName: mnFirst,
	}
	b.replaceFields[tnSecond] = &replaceField{
		tableName: tnSecond,
		modelName: mnSecond,
	}

	return b
}

// Retrieve the subquery info, including subquery itself and associated args
func getSubQuery(val interface{}) (string, *orderedmap.OrderedMap[string, interface{}]) {
	if isTarget(val) {
		ref := reflect.ValueOf(val)
		query, _ := ref.MethodByName("AllQuery").Call([]reflect.Value{})[0].Interface().(string)
		argMap, _ := ref.MethodByName("GetArgs").Call([]reflect.Value{})[0].Interface().(*orderedmap.OrderedMap[string, interface{}])
		return query, argMap
	}

	return "", nil
}

// Retrieve the table and module name of given struct
func getQuerySetInfo(val interface{}) (string, string) {
	if isTarget(val) {
		ref := reflect.ValueOf(val)
		rets := ref.MethodByName("GetModel").Call([]reflect.Value{})
		if rets != nil {
			tname, _ := rets[0].Interface().(string)
			mname, _ := rets[1].Interface().(string)
			return tname, mname
		}
	} else if reflect.TypeOf(val).Kind() == reflect.Struct {
		mname := reflect.TypeOf(val).Name()
		tname := mname
		ref := reflect.ValueOf(val)
		for i := 0; i < ref.NumField(); i++ {
			field := ref.Type().Field(i)
			if strings.Compare(field.Name, PikaMetadataTableName) == 0 {
				tname = field.Tag.Get("pika")
				break
			}
		}
		return tname, mname
	}

	return "", ""
}

// Check whether the val is type of QuerySet[T]
func isTarget(val interface{}) bool {
	if reflect.TypeOf(val).Kind() == reflect.Ptr {
		ref := reflect.ValueOf(val)
		return !(ref.MethodByName("GetModel").IsZero())
	}
	return false
}

// Replace the old place holder with new ones
//
//nolint:predeclared
func replacePlaceHolder(query string, old, new []int) string {
	if len(old) != len(new) {
		return query
	}

	for idx := range old {
		query = strings.Replace(query, fmt.Sprintf("$%d", old[idx]), fmt.Sprintf("$%d", new[idx]), 1)
	}

	return query
}

// Generate int slice in range [start, start + length -1]
func generateRangeSlice(start, length int) *[]int {
	if length == 1 {
		return &[]int{start}
	}

	ret := make([]int, length)
	for idx := 0; idx < length; idx++ {
		ret[idx] = start + idx
	}
	return &ret
}
