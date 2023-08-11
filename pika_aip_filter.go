// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	"github.com/pkg/errors"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.ciq.dev/pika/parser"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The goal of this AIP Filter extension is to be able to parse
// grammar like the one used in AIP-160.
// This extension uses Antlr generated parser to parse the
// filter string.
// The extension returns a QuerySet that can be used to query the
// database.
// The filters on the QuerySet are applied in the order they are
// given.

type AIPFilter[T any] struct {
	QuerySet[T]
}

type AIPFilterIdentifier struct {
	// Value aliases are used to map a value to a different value.
	// Mostly useful for enums, where for example the values
	// STAGE_STATUS_FAILED, FAILED, FaIlEd, etc. should all be
	// mapped to the same value.
	// This means that the alias value is case insensitive.
	// Make sure to convert to lower case if key is a string.
	ValueAliases map[any]any

	// AcceptedTypes is a list of types that are accepted for this
	// identifier.
	// If empty, all types are accepted.
	// The value should be in antlrValues
	AcceptedTypes []int

	// AcceptableValues is a list of values that are accepted for this
	// identifier.
	// If empty, all values are accepted.
	AcceptedValues []any

	// Column name is the name of the column in the database.
	// If empty, the identifier is used as the column name.
	ColumnName string

	// IsRepeated is true if the identifier is a repeated field.
	// This is used to determine how to apply the filter.
	IsRepeated bool
}

type AIPFilterOptions struct {
	// Identifiers are additional configuration for specific identifiers.
	Identifiers map[string]AIPFilterIdentifier

	// AcceptableIdentifiers is a list of identifiers that are allowed
	AcceptableIdentifiers []string
}

func (a AIPFilterOptions) verifyOrderBy(orderBy string) ([]string, error) {
	// If empty, return the QuerySet as is.
	if orderBy == "" {
		return []string{}, nil
	}

	// Split the orderBy string
	orderBySplit := strings.Split(orderBy, ",")

	// New order list with proper column names
	var newOrderBy []string

	// Verify that all identifiers are acceptable
	for _, fullIdentifier := range orderBySplit {
		// Check if there is an asc/desc suffix
		idents := strings.Split(fullIdentifier, " ")
		identifier := idents[0]
		sort := "asc"
		if len(idents) > 1 {
			sort = strings.ToLower(idents[1])
			// Check if the suffix is valid
			if sort != "asc" && sort != "desc" {
				return nil, fmt.Errorf("invalid suffix %s for identifier %s", sort, identifier)
			}
		}
		prefix := ""
		if sort == "desc" {
			prefix = "-"
		}

		// Verify that the identifier is acceptable
		if !contains(a.AcceptableIdentifiers, identifier) {
			return nil, fmt.Errorf("identifier %s is not acceptable", identifier)
		}

		// Check if column name is defined
		if ident, ok := a.Identifiers[identifier]; ok {
			if ident.ColumnName != "" {
				identifier = ident.ColumnName
			}
		}

		newOrderBy = append(newOrderBy, prefix+identifier)
	}

	return newOrderBy, nil
}

func NewAIPFilter[T any]() *AIPFilter[T] {
	return &AIPFilter[T]{}
}

func (a *AIPFilter[T]) parseFilter(filter string) (*parser.FilterLexer, error) {
	// If the filter is empty, return the QuerySet as is.
	if filter == "" {
		return nil, errors.New("filter string is empty")
	}

	input := antlr.NewInputStream(filter)
	lexer := parser.NewFilterLexer(input)

	return lexer, nil
}

// AIPFilter parses the filter string from a gRPC request and
// returns a QuerySet that can be used to query the database.
func (a *AIPFilter[T]) aip160(b QuerySet[T], filter string, options AIPFilterOptions) (QuerySet[T], error) {
	// If empty, return the QuerySet as is.
	if filter == "" {
		return b, nil
	}

	// Verify options
	if options.Identifiers == nil {
		options.Identifiers = map[string]AIPFilterIdentifier{}
	}
	for identifier, opts := range options.Identifiers {
		// Verify that accepted types is in antlrValues
		for _, acceptedType := range opts.AcceptedTypes {
			if _, ok := antlrValues[acceptedType]; !ok {
				return nil, errors.Errorf("invalid accepted type %d for identifier %s", acceptedType, identifier)
			}
		}

		// Verify that string type keys in value aliases are lower case
		for key := range opts.ValueAliases {
			if _, ok := key.(string); ok {
				if key != strings.ToLower(key.(string)) {
					return nil, errors.Errorf("value alias key %s for identifier %s is not lower case", key, identifier)
				}
			}
		}
	}

	lexer, err := a.parseFilter(filter)
	if err != nil {
		return nil, err
	}

	// Walk the filter tree and apply the filters to the QuerySet.
	states := []*pikaAip160State{
		{
			activeParens:  true,
			initParens:    true,
			filterContent: []*pikaFiltering{},
			args:          orderedmap.New[string, any](),
		},
	}
	i := 0
	for {
		activeState := states[i]

		t := lexer.NextToken()
		if t.GetTokenType() == antlr.TokenEOF {
			// If we had an activeParens (usually no parens, is an active parens too)
			// We add the state to the QuerySet
			// If it's empty, it's skipped anyways
			if activeState.activeParens && activeState.activeExpr != nil {
				// If we have an active expression, add it to the filter content
				activeState.filterContent = append(activeState.filterContent, activeState.activeExpr)
				activeState.activeExpr = nil
			}
			break
		}

		tokenType := t.GetTokenType()

		// Decide what to do with token
		switch tokenType {
		// Left parenthesis.
		// This means that the next expression is a subexpression.
		// And will be grouped in the same Filter call
		case parser.FilterLexerLPAREN:
			// We currently don't support nested expressions like this
			// (a = 1 AND (b = 2 OR c = 3))
			if !activeState.initParens && activeState.activeParens {
				return nil, fmt.Errorf("nested expressions are not supported")
			}

			// Disallow combined expression for values
			if activeState.activeIdentifier != "" {
				return nil, fmt.Errorf("cannot combine multiple values in subexpression")
			}

			if activeState.activeParens && !activeState.initParens {
				// Nested expressions are marked with innerParens
				innerState := &pikaAip160State{
					activeParens:  true,
					innerParens:   true,
					filterContent: []*pikaFiltering{},
					args:          orderedmap.New[string, any](),
				}

				states = append(states, innerState)
				i++
			} else {
				activeState.activeParens = true
				activeState.initParens = false
				activeState.forceNot = activeState.activeNot
			}
			continue
		// Right parenthesis.
		// Closing previous parenthesis.
		case parser.FilterLexerRPAREN:
			activeState.activeParens = false
		// If whitespace, ignore.
		case parser.FilterLexerWHITESPACE:
			continue
		// If OR, enable OR mode.
		case parser.FilterLexerOR:
			// If it's already active, we need to add a hint to force innerOr
			// If activeOr is true, we need to add a hint instead
			activeState.activeOperator += HintOr
			activeState.activeOr = true
			continue
		// If AND, disable OR mode.
		case parser.FilterLexerAND:
			// We support multiple hints, since AND is default, we just
			// need to force it if OR is activated
			activeState.activeOr = false
			activeState.activeOperator += HintAnd
			continue
		// If NOT, enable NOT mode.
		case parser.FilterLexerNOT,
			parser.FilterLexerMINUS:
			activeState.activeNot = true
			continue
		}

		// Check if operator
		if activeState.activeNot {
			// Manually handle has operator for array fields
			if tokenType == parser.FilterLexerCOLON {
				if x, ok := options.Identifiers[activeState.activeIdentifier]; ok {
					if x.IsRepeated {
						activeState.activeOperator = HintNotIn
						continue
					}
				}
			}

			// If NOT, check if operator
			if x, ok := antlrOperatorsNot[tokenType]; ok {
				// If operator, set the current operator.
				newVal := fmt.Sprintf("%s%s", x, activeState.activeOperator)
				activeState.activeOperator = newVal
				continue
			}
		} else {
			// Manually handle has operator for array fields
			if tokenType == parser.FilterLexerCOLON {
				if x, ok := options.Identifiers[activeState.activeIdentifier]; ok {
					if x.IsRepeated {
						activeState.activeOperator = HintIn
						continue
					}
				}
			}
			if x, ok := antlrOperators[tokenType]; ok {
				// If operator, set the current operator.
				newVal := fmt.Sprintf("%s%s", x, activeState.activeOperator)
				activeState.activeOperator = newVal
				continue
			}
		}

		// Check values
		lacksValue := activeState.activeValue == nil

		switch tokenType {
		case parser.FilterLexerSTRING:
			stringVal := strings.Trim(t.GetText(), "\"")
			// If the value contains a * add a like hint
			if strings.Contains(stringVal, "*") {
				activeState.activeOperator += HintLike
				stringVal = strings.ReplaceAll(stringVal, "*", "%")
			}
			activeState.activeValue = stringVal
			activeState.activeValueType = parser.FilterLexerSTRING
		case parser.FilterLexerDURATION:
			var r durationpb.Duration
			val := strconv.Quote(strings.Trim(t.GetText(), "\""))
			err := protojson.Unmarshal([]byte(val), &r)
			if err != nil {
				return nil, err
			}
			activeState.activeValue = r.AsDuration()
			activeState.activeValueType = parser.FilterLexerDURATION
		case parser.FilterLexerTIMESTAMP:
			var r timestamppb.Timestamp
			val := strconv.Quote(strings.Trim(t.GetText(), "\""))
			err := protojson.Unmarshal([]byte(val), &r)
			if err != nil {
				return nil, err
			}
			activeState.activeValue = r.AsTime()
			activeState.activeValueType = parser.FilterLexerTIMESTAMP
		case parser.FilterLexerNUM_FLOAT:
			f64, err := strconv.ParseFloat(t.GetText(), 64)
			if err != nil {
				return nil, err
			}
			activeState.activeValue = f64
			activeState.activeValueType = parser.FilterLexerNUM_FLOAT
		case parser.FilterLexerNUM_INT:
			i64, err := strconv.ParseInt(t.GetText(), 10, 64)
			if err != nil {
				return nil, err
			}
			activeState.activeValue = i64
			activeState.activeValueType = parser.FilterLexerNUM_INT
		case parser.FilterLexerNUM_UINT:
			u64, err := strconv.ParseUint(t.GetText(), 10, 64)
			if err != nil {
				return nil, err
			}
			activeState.activeValue = u64
			activeState.activeValueType = parser.FilterLexerNUM_UINT
		case parser.FilterLexerTRUE:
			activeState.activeValue = true
			activeState.activeValueType = parser.FilterLexerTRUE
		case parser.FilterLexerFALSE:
			activeState.activeValue = false
			activeState.activeValueType = parser.FilterLexerFALSE
		case parser.FilterLexerNULL:
			// Null is an operator and a value
			setOp := HintIsNull
			if activeState.activeNot {
				setOp = HintIsNotNull
			}

			newVal := fmt.Sprintf("%s%s", setOp, activeState.activeOperator)
			activeState.activeOperator = newVal

			activeState.activeValue = setOp
			activeState.activeValueType = parser.FilterLexerNULL
		}

		// Check if value was set in previous switch
		if lacksValue && activeState.activeValue != nil {
			cnf, ok := options.Identifiers[activeState.activeIdentifier]
			if ok {
				// Check if we have a value alias
				val := activeState.activeValue

				// If string, make it lowercase
				if s, ok := val.(string); ok {
					val = strings.ToLower(s)
				}

				alias, ok := cnf.ValueAliases[val]
				if ok {
					activeState.activeValue = alias

					// If alias is a string, we need to set the type to string
					if _, ok := alias.(string); ok {
						activeState.activeValueType = parser.FilterLexerSTRING
					} else if x, ok := alias.(bool); ok {
						// If alias is a bool, we need to set the type to bool
						if x {
							activeState.activeValueType = parser.FilterLexerTRUE
						} else {
							activeState.activeValueType = parser.FilterLexerFALSE
						}
					} else if _, ok := alias.(*durationpb.Duration); ok {
						// If alias is a duration, we need to set the type to duration
						activeState.activeValueType = parser.FilterLexerDURATION
					} else if _, ok := alias.(*timestamppb.Timestamp); ok {
						// If alias is a timestamp, we need to set the type to timestamp
						activeState.activeValueType = parser.FilterLexerTIMESTAMP
					} else if _, ok := alias.(float64); ok {
						// If alias is a float64, we need to set the type to float64
						activeState.activeValueType = parser.FilterLexerNUM_FLOAT
					} else if _, ok := alias.(int64); ok {
						// If alias is a int64, we need to set the type to int64
						activeState.activeValueType = parser.FilterLexerNUM_INT
					} else if _, ok := alias.(uint64); ok {
						// If alias is a uint64, we need to set the type to uint64
						activeState.activeValueType = parser.FilterLexerNUM_UINT
					} else {
						return nil, fmt.Errorf("unknown alias type %T", alias)
					}
				}

				// Verify if type matches any accepted types
				// If not, return error
				if len(cnf.AcceptedTypes) > 0 {
					isOk := false
					for _, t := range cnf.AcceptedTypes {
						if t == activeState.activeValueType {
							isOk = true
							break
						}
					}
					if !isOk {
						return nil, fmt.Errorf("type %s is not accepted for identifier %s", lexer.SymbolicNames[activeState.activeValueType], activeState.activeIdentifier)
					}
				}

				// Verify if value matches any accepted values
				// If not, return error
				if len(cnf.AcceptedValues) > 0 {
					isOk := false
					for _, v := range cnf.AcceptedValues {
						if v == activeState.activeValue {
							isOk = true
							break
						}
					}
					if !isOk {
						return nil, fmt.Errorf("value %v is not accepted for identifier %s", activeState.activeValue, activeState.activeIdentifier)
					}
				}
			}
		}

		// Check if value, if so add filter with value.
		if tokenType == parser.FilterLexerIDENTIFIER {
			// If we already have an identifier, then this is a value
			if activeState.activeIdentifier == "" {
				activeState.activeIdentifier = t.GetText()
				continue
			}
			return nil, fmt.Errorf("unexpected identifier %s", t.GetText())
		}

		if activeState.activeOperator != "" && activeState.activeIdentifier != "" && activeState.activeValue != nil {
			operator := activeState.activeOperator
			if operator == "" {
				return nil, fmt.Errorf("missing operator")
			}
			if activeState.activeIdentifier == "" {
				return nil, fmt.Errorf("missing identifier")
			}

			// Check if AcceptableIdentifiers are set, if so check if identifier is valid
			if len(options.AcceptableIdentifiers) > 0 {
				if !contains(options.AcceptableIdentifiers, activeState.activeIdentifier) {
					return nil, fmt.Errorf("identifier %s is not allowed", activeState.activeIdentifier)
				}
			}

			// If operator is __eq, then make it empty
			// Default action is __eq already
			operator = strings.ReplaceAll(operator, "__eq", "")

			// Add filter to the state.
			if activeState.activeExpr == nil {
				activeState.activeExpr = &pikaFiltering{
					entries: orderedmap.New[string, string](),
					or:      activeState.activeOr,
					innerOr: false,
				}
			}

			dbColumn := activeState.activeIdentifier
			// Check if we have a column name override
			cnf, ok := options.Identifiers[activeState.activeIdentifier]
			if ok {
				if cnf.ColumnName != "" {
					// If we have a column name override, use that instead
					dbColumn = cnf.ColumnName
				}
			}

			// Add filter to current state
			key := dbColumn
			if operator != "" {
				key = fmt.Sprintf("%s%s", dbColumn, operator)
			}
			key = findEmptyForKey(key, activeState.activeExpr.entries)
			prefixCount := strings.Count(key, "!")
			suffix := "_aip160_"
			if prefixCount > 0 {
				suffix += strconv.Itoa(prefixCount)
			}
			argKey := fmt.Sprintf("%s%s", cleanKey(key), suffix)
			value := fmt.Sprintf(":%s", argKey)

			// For AIP-160 purposes, if the operator has HintILike, then we need to
			// wrap the value in % to match wildcard
			if strings.Contains(operator, HintILike) || strings.Contains(operator, HintNotILike) {
				value = fmt.Sprintf("%%%s%%", value)
			}

			isNullPrefix := strings.HasPrefix(operator, HintIsNull)
			isNotNullPrefix := strings.HasPrefix(operator, HintIsNotNull)
			if isNullPrefix || isNotNullPrefix {
				value = "true"
			}
			activeState.activeExpr.entries.Set(key, value)

			// Add arg
			if !isNullPrefix && !isNotNullPrefix {
				activeState.args.Set(argKey, activeState.activeValue)
			}

			activeState.activeIdentifier = ""
			activeState.activeOperator = ""
			activeState.activeValue = nil
			if !activeState.forceNot {
				activeState.activeNot = false
			}
		}

		// If not active parens, we need to create a new state.
		if !activeState.activeParens {
			if activeState.innerParens {
				i--
				activeState.innerParens = false
			} else {
				i++
				states = append(states, &pikaAip160State{
					activeParens:  false,
					filterContent: []*pikaFiltering{},
					args:          orderedmap.New[string, any](),
				})

				if activeState.activeExpr != nil {
					// If we have an active expression, add it to the filter content
					activeState.filterContent = append(activeState.filterContent, activeState.activeExpr)
					activeState.activeExpr = nil
				}
			}
		}
	}

	// Convert states to filters
	for _, state := range states {
		// No filters, skip
		if len(state.filterContent) == 0 {
			continue
		}

		// If args, then add them
		if state.args.Len() > 0 {
			b.Args(state.args)
		}

		// Each filterContent should be grouped together in same filter call
		// This ensures that parantheses are respected.
		for _, filter := range state.filterContent {
			var filterQueries []string

			for pair := filter.entries.Oldest(); pair != nil; pair = pair.Next() {
				key := pair.Key
				value := pair.Value

				filterQueries = append(filterQueries, fmt.Sprintf("%s=%s", cleanKey(key), value))
			}

			if filter.or {
				b.FilterOr(filterQueries...)
			} else {
				b.Filter(filterQueries...)
			}
		}
	}

	return b, nil
}
