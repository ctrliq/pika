// Code generated from java-escape by ANTLR 4.11.1. DO NOT EDIT.

package parser // Filter
import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

// FilterListener is a complete listener for a parse tree produced by FilterParser.
type FilterListener interface {
	antlr.ParseTreeListener

	// EnterFilter is called when entering the filter production.
	EnterFilter(c *FilterContext)

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterFactor is called when entering the factor production.
	EnterFactor(c *FactorContext)

	// EnterTerm is called when entering the term production.
	EnterTerm(c *TermContext)

	// EnterSimple is called when entering the simple production.
	EnterSimple(c *SimpleContext)

	// EnterRestriction is called when entering the restriction production.
	EnterRestriction(c *RestrictionContext)

	// EnterComparable is called when entering the comparable production.
	EnterComparable(c *ComparableContext)

	// EnterMember is called when entering the member production.
	EnterMember(c *MemberContext)

	// EnterFunction is called when entering the function production.
	EnterFunction(c *FunctionContext)

	// EnterComparator is called when entering the comparator production.
	EnterComparator(c *ComparatorContext)

	// EnterComposite is called when entering the composite production.
	EnterComposite(c *CompositeContext)

	// EnterInt is called when entering the Int production.
	EnterInt(c *IntContext)

	// EnterDouble is called when entering the Double production.
	EnterDouble(c *DoubleContext)

	// EnterUint is called when entering the Uint production.
	EnterUint(c *UintContext)

	// EnterString is called when entering the String production.
	EnterString(c *StringContext)

	// EnterDuration is called when entering the Duration production.
	EnterDuration(c *DurationContext)

	// EnterTimestamp is called when entering the Timestamp production.
	EnterTimestamp(c *TimestampContext)

	// EnterBoolTrue is called when entering the BoolTrue production.
	EnterBoolTrue(c *BoolTrueContext)

	// EnterBoolFalse is called when entering the BoolFalse production.
	EnterBoolFalse(c *BoolFalseContext)

	// EnterNull is called when entering the Null production.
	EnterNull(c *NullContext)

	// EnterField is called when entering the field production.
	EnterField(c *FieldContext)

	// EnterName is called when entering the name production.
	EnterName(c *NameContext)

	// EnterArgList is called when entering the argList production.
	EnterArgList(c *ArgListContext)

	// EnterArg is called when entering the arg production.
	EnterArg(c *ArgContext)

	// EnterKeyword is called when entering the keyword production.
	EnterKeyword(c *KeywordContext)

	// ExitFilter is called when exiting the filter production.
	ExitFilter(c *FilterContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitFactor is called when exiting the factor production.
	ExitFactor(c *FactorContext)

	// ExitTerm is called when exiting the term production.
	ExitTerm(c *TermContext)

	// ExitSimple is called when exiting the simple production.
	ExitSimple(c *SimpleContext)

	// ExitRestriction is called when exiting the restriction production.
	ExitRestriction(c *RestrictionContext)

	// ExitComparable is called when exiting the comparable production.
	ExitComparable(c *ComparableContext)

	// ExitMember is called when exiting the member production.
	ExitMember(c *MemberContext)

	// ExitFunction is called when exiting the function production.
	ExitFunction(c *FunctionContext)

	// ExitComparator is called when exiting the comparator production.
	ExitComparator(c *ComparatorContext)

	// ExitComposite is called when exiting the composite production.
	ExitComposite(c *CompositeContext)

	// ExitInt is called when exiting the Int production.
	ExitInt(c *IntContext)

	// ExitDouble is called when exiting the Double production.
	ExitDouble(c *DoubleContext)

	// ExitUint is called when exiting the Uint production.
	ExitUint(c *UintContext)

	// ExitString is called when exiting the String production.
	ExitString(c *StringContext)

	// ExitDuration is called when exiting the Duration production.
	ExitDuration(c *DurationContext)

	// ExitTimestamp is called when exiting the Timestamp production.
	ExitTimestamp(c *TimestampContext)

	// ExitBoolTrue is called when exiting the BoolTrue production.
	ExitBoolTrue(c *BoolTrueContext)

	// ExitBoolFalse is called when exiting the BoolFalse production.
	ExitBoolFalse(c *BoolFalseContext)

	// ExitNull is called when exiting the Null production.
	ExitNull(c *NullContext)

	// ExitField is called when exiting the field production.
	ExitField(c *FieldContext)

	// ExitName is called when exiting the name production.
	ExitName(c *NameContext)

	// ExitArgList is called when exiting the argList production.
	ExitArgList(c *ArgListContext)

	// ExitArg is called when exiting the arg production.
	ExitArg(c *ArgContext)

	// ExitKeyword is called when exiting the keyword production.
	ExitKeyword(c *KeywordContext)
}
