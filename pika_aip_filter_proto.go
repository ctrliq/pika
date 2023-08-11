// SPDX-FileCopyrightText: Copyright (c) 2023, Ctrl IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package pika

import (
	"strings"

	"github.com/iancoleman/strcase"
	"go.ciq.dev/pika/parser"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	protoKindToLexer = map[protoreflect.Kind]int{
		protoreflect.StringKind: parser.FilterLexerSTRING,
		protoreflect.FloatKind:  parser.FilterLexerNUM_FLOAT,
		protoreflect.Int32Kind:  parser.FilterLexerNUM_INT,
		protoreflect.Int64Kind:  parser.FilterLexerNUM_INT,
		protoreflect.Uint32Kind: parser.FilterLexerNUM_UINT,
		protoreflect.Uint64Kind: parser.FilterLexerNUM_UINT,
	}
	protoMessageKindToLexer = map[string]int{
		"google.protobuf.StringValue": parser.FilterLexerSTRING,
		"google.protobuf.Duration":    parser.FilterLexerDURATION,
		"google.protobuf.Timestamp":   parser.FilterLexerTIMESTAMP,
		"google.protobuf.DoubleValue": parser.FilterLexerNUM_FLOAT,
		"google.protobuf.FloatValue":  parser.FilterLexerNUM_FLOAT,
		"google.protobuf.Int32Value":  parser.FilterLexerNUM_INT,
		"google.protobuf.Int64Value":  parser.FilterLexerNUM_INT,
		"google.protobuf.UInt32Value": parser.FilterLexerNUM_UINT,
		"google.protobuf.UInt64Value": parser.FilterLexerNUM_UINT,
	}
)

type ProtoReflectOptions struct {
	// Exclude is a list of field names to exclude from the filter
	// Uses proto name always, not JSON name
	Exclude []string

	// ColumnName is a function that returns the column name for a given field
	// name. If not provided, the field name is used.
	ColumnName func(string) string
}

func protoReflect(m proto.Message, opts ProtoReflectOptions) AIPFilterOptions {
	res := AIPFilterOptions{
		Identifiers:           map[string]AIPFilterIdentifier{},
		AcceptableIdentifiers: []string{},
	}

	fields := m.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		// Get field from message
		fd := fields.Get(i)

		// Get name, use JSON name if available
		name := string(fd.Name())
		// Skip if excluded
		if contains(opts.Exclude, name) {
			continue
		}
		if fd.JSONName() != "" {
			name = fd.JSONName()
		}

		// Now let's configure the identifiers map with acceptable value and
		// potential aliases for enum
		ident := AIPFilterIdentifier{
			ValueAliases:   map[any]any{},
			AcceptedTypes:  []int{},
			AcceptedValues: []any{},
			IsRepeated:     false,
		}

		// Check if repeated
		if fd.Cardinality() == protoreflect.Repeated {
			ident.IsRepeated = true
		}

		// Check and add type
		// If type is message and not a supported wrapper type, skip
		switch fd.Kind() {
		case protoreflect.StringKind,
			protoreflect.FloatKind,
			protoreflect.Int32Kind,
			protoreflect.Int64Kind,
			protoreflect.Uint32Kind,
			protoreflect.Uint64Kind:
			ident.AcceptedTypes = append(ident.AcceptedTypes, protoKindToLexer[fd.Kind()])
		case protoreflect.BoolKind:
			ident.AcceptedTypes = append(ident.AcceptedTypes, parser.FilterLexerTRUE, parser.FilterLexerFALSE)
		case protoreflect.MessageKind:
			// Bool wrappers need two types
			if fd.Message().FullName() == "google.protobuf.BoolValue" {
				ident.AcceptedTypes = append(ident.AcceptedTypes, parser.FilterLexerTRUE, parser.FilterLexerFALSE, parser.FilterLexerNULL)
			} else {
				// Check if it's a supported wrapper type
				if lexer, ok := protoMessageKindToLexer[string(fd.Message().FullName())]; ok {
					ident.AcceptedTypes = append(ident.AcceptedTypes, lexer)

					// All wrappers can be nil
					ident.AcceptedTypes = append(ident.AcceptedTypes, parser.FilterLexerNULL)
				} else {
					// Skip this field
					continue
				}
			}
		case protoreflect.EnumKind:
			// Add all enum values as aliases
			for i := 0; i < fd.Enum().Values().Len(); i++ {
				enumName := string(fd.Enum().Values().Get(i).Name())
				enumReverseSplit := strings.Split(enumName, "_")
				for i := 0; i < len(enumReverseSplit)/2; i++ {
					j := len(enumReverseSplit) - i - 1
					enumReverseSplit[i], enumReverseSplit[j] = enumReverseSplit[j], enumReverseSplit[i]
				}
				lastValue := enumReverseSplit[0]

				// Both the name and the last value are acceptable
				// Only add last value if it's not the same as the name
				num := int64(fd.Enum().Values().Get(i).Number())
				ident.ValueAliases[strings.ToLower(enumName)] = num
				if lastValue != enumName {
					ident.ValueAliases[strings.ToLower(lastValue)] = num
				}

				// Add num as acceptable value
				ident.AcceptedValues = append(ident.AcceptedValues, num)
			}
			// Add the enum value as a type
			ident.AcceptedTypes = append(ident.AcceptedTypes, parser.FilterLexerNUM_INT)
		}

		// Add column name
		if opts.ColumnName != nil {
			ident.ColumnName = opts.ColumnName(name)
		} else {
			ident.ColumnName = strcase.ToSnake(name)
		}

		// Add to identifiers
		res.Identifiers[name] = ident
		res.AcceptableIdentifiers = append(res.AcceptableIdentifiers, name)
	}

	return res
}

func ProtoReflect(m proto.Message) AIPFilterOptions {
	return protoReflect(m, ProtoReflectOptions{})
}

func ProtoReflectWithOpts(m proto.Message, opts ProtoReflectOptions) AIPFilterOptions {
	return protoReflect(m, opts)
}
