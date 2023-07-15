// Code generated from java-escape by ANTLR 4.11.1. DO NOT EDIT.

package parser // Filter
import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

// BaseFilterListener is a complete listener for a parse tree produced by FilterParser.
type BaseFilterListener struct{}

var _ FilterListener = &BaseFilterListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseFilterListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseFilterListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseFilterListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseFilterListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterFilter is called when production filter is entered.
func (s *BaseFilterListener) EnterFilter(ctx *FilterContext) {}

// ExitFilter is called when production filter is exited.
func (s *BaseFilterListener) ExitFilter(ctx *FilterContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseFilterListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseFilterListener) ExitExpression(ctx *ExpressionContext) {}

// EnterFactor is called when production factor is entered.
func (s *BaseFilterListener) EnterFactor(ctx *FactorContext) {}

// ExitFactor is called when production factor is exited.
func (s *BaseFilterListener) ExitFactor(ctx *FactorContext) {}

// EnterTerm is called when production term is entered.
func (s *BaseFilterListener) EnterTerm(ctx *TermContext) {}

// ExitTerm is called when production term is exited.
func (s *BaseFilterListener) ExitTerm(ctx *TermContext) {}

// EnterSimple is called when production simple is entered.
func (s *BaseFilterListener) EnterSimple(ctx *SimpleContext) {}

// ExitSimple is called when production simple is exited.
func (s *BaseFilterListener) ExitSimple(ctx *SimpleContext) {}

// EnterRestriction is called when production restriction is entered.
func (s *BaseFilterListener) EnterRestriction(ctx *RestrictionContext) {}

// ExitRestriction is called when production restriction is exited.
func (s *BaseFilterListener) ExitRestriction(ctx *RestrictionContext) {}

// EnterComparable is called when production comparable is entered.
func (s *BaseFilterListener) EnterComparable(ctx *ComparableContext) {}

// ExitComparable is called when production comparable is exited.
func (s *BaseFilterListener) ExitComparable(ctx *ComparableContext) {}

// EnterMember is called when production member is entered.
func (s *BaseFilterListener) EnterMember(ctx *MemberContext) {}

// ExitMember is called when production member is exited.
func (s *BaseFilterListener) ExitMember(ctx *MemberContext) {}

// EnterFunction is called when production function is entered.
func (s *BaseFilterListener) EnterFunction(ctx *FunctionContext) {}

// ExitFunction is called when production function is exited.
func (s *BaseFilterListener) ExitFunction(ctx *FunctionContext) {}

// EnterComparator is called when production comparator is entered.
func (s *BaseFilterListener) EnterComparator(ctx *ComparatorContext) {}

// ExitComparator is called when production comparator is exited.
func (s *BaseFilterListener) ExitComparator(ctx *ComparatorContext) {}

// EnterComposite is called when production composite is entered.
func (s *BaseFilterListener) EnterComposite(ctx *CompositeContext) {}

// ExitComposite is called when production composite is exited.
func (s *BaseFilterListener) ExitComposite(ctx *CompositeContext) {}

// EnterInt is called when production Int is entered.
func (s *BaseFilterListener) EnterInt(ctx *IntContext) {}

// ExitInt is called when production Int is exited.
func (s *BaseFilterListener) ExitInt(ctx *IntContext) {}

// EnterDouble is called when production Double is entered.
func (s *BaseFilterListener) EnterDouble(ctx *DoubleContext) {}

// ExitDouble is called when production Double is exited.
func (s *BaseFilterListener) ExitDouble(ctx *DoubleContext) {}

// EnterUint is called when production Uint is entered.
func (s *BaseFilterListener) EnterUint(ctx *UintContext) {}

// ExitUint is called when production Uint is exited.
func (s *BaseFilterListener) ExitUint(ctx *UintContext) {}

// EnterString is called when production String is entered.
func (s *BaseFilterListener) EnterString(ctx *StringContext) {}

// ExitString is called when production String is exited.
func (s *BaseFilterListener) ExitString(ctx *StringContext) {}

// EnterDuration is called when production Duration is entered.
func (s *BaseFilterListener) EnterDuration(ctx *DurationContext) {}

// ExitDuration is called when production Duration is exited.
func (s *BaseFilterListener) ExitDuration(ctx *DurationContext) {}

// EnterTimestamp is called when production Timestamp is entered.
func (s *BaseFilterListener) EnterTimestamp(ctx *TimestampContext) {}

// ExitTimestamp is called when production Timestamp is exited.
func (s *BaseFilterListener) ExitTimestamp(ctx *TimestampContext) {}

// EnterBoolTrue is called when production BoolTrue is entered.
func (s *BaseFilterListener) EnterBoolTrue(ctx *BoolTrueContext) {}

// ExitBoolTrue is called when production BoolTrue is exited.
func (s *BaseFilterListener) ExitBoolTrue(ctx *BoolTrueContext) {}

// EnterBoolFalse is called when production BoolFalse is entered.
func (s *BaseFilterListener) EnterBoolFalse(ctx *BoolFalseContext) {}

// ExitBoolFalse is called when production BoolFalse is exited.
func (s *BaseFilterListener) ExitBoolFalse(ctx *BoolFalseContext) {}

// EnterNull is called when production Null is entered.
func (s *BaseFilterListener) EnterNull(ctx *NullContext) {}

// ExitNull is called when production Null is exited.
func (s *BaseFilterListener) ExitNull(ctx *NullContext) {}

// EnterField is called when production field is entered.
func (s *BaseFilterListener) EnterField(ctx *FieldContext) {}

// ExitField is called when production field is exited.
func (s *BaseFilterListener) ExitField(ctx *FieldContext) {}

// EnterName is called when production name is entered.
func (s *BaseFilterListener) EnterName(ctx *NameContext) {}

// ExitName is called when production name is exited.
func (s *BaseFilterListener) ExitName(ctx *NameContext) {}

// EnterArgList is called when production argList is entered.
func (s *BaseFilterListener) EnterArgList(ctx *ArgListContext) {}

// ExitArgList is called when production argList is exited.
func (s *BaseFilterListener) ExitArgList(ctx *ArgListContext) {}

// EnterArg is called when production arg is entered.
func (s *BaseFilterListener) EnterArg(ctx *ArgContext) {}

// ExitArg is called when production arg is exited.
func (s *BaseFilterListener) ExitArg(ctx *ArgContext) {}

// EnterKeyword is called when production keyword is entered.
func (s *BaseFilterListener) EnterKeyword(ctx *KeywordContext) {}

// ExitKeyword is called when production keyword is exited.
func (s *BaseFilterListener) ExitKeyword(ctx *KeywordContext) {}
