// SPDX-FileCopyrightText: Copyright (c) 2023-2025, CTRL IQ, Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0

//go:generate antlr -Dlanguage=Go -o . Filter.g4

// Package parser provides ANTLR-generated parsers for filter expressions.
// This package contains generated code for parsing AIP-160 compliant filter syntax.
package parser
