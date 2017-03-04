package ast

import (
	"fmt"
	"strings"
)

// printOriginalVistor implements the Visitor interface to print a AST.
type printOriginalVistor struct {
	buf   string
	depth int

	original bool
	inBlock  bool
}

func newPrintOriginalVisitor() *printOriginalVistor {
	return &printOriginalVistor{}
}

// Print returns a string representation of given AST, that can be used for debugging purpose.
func PrintOriginal(node Node) string {
	visitor := newPrintOriginalVisitor()
	node.Accept(visitor)
	return visitor.output()
}

func (v *printOriginalVistor) output() string {
	return v.buf
}

func (v *printOriginalVistor) indent() {
	for i := 0; i < v.depth; {
		v.buf += "  "
		i++
	}
}

func (v *printOriginalVistor) str(val string) {
	v.buf += val
}

func (v *printOriginalVistor) nl() {
	v.str("\n")
}

func (v *printOriginalVistor) line(val string) {
	v.indent()
	v.str(val)
	v.nl()
}

//
// Visitor interface
//

// Statements

// VisitProgram implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitProgram(node *Program) interface{} {
	if len(node.BlockParams) > 0 {
		v.line("BLOCK PARAMS: [ " + strings.Join(node.BlockParams, " ") + " ]")
	}

	for _, n := range node.Body {
		n.Accept(v)
	}

	return nil
}

// VisitMustache implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitMustache(node *MustacheStatement) interface{} {
	v.str("{{")
	node.Expression.Accept(v)
	v.str("}}")
	return nil
}

// VisitBlock implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitBlock(node *BlockStatement) interface{} {
	v.inBlock = true
	// v.str("{{#")
	v.depth++

	v.str("{{#")
	node.Expression.Accept(v)
	v.str("}}")

	if node.Program != nil {
		// v.line("PROGRAM:")
		v.depth++
		node.Program.Accept(v)
		v.depth--
	}

	if node.Inverse != nil {
		// if node.Program != nil {
		// 	v.depth++
		// }

		v.str("{{else}}")
		v.depth++
		node.Inverse.Accept(v)
		v.depth--

		// if node.Program != nil {
		// 	v.depth--
		// }
	}
	if node.Expression.Path != nil {
		if p, ok := node.Expression.Path.(*PathExpression); ok {
			v.str(fmt.Sprintf("{{/%s}}", p.Original))
		}
	}

	v.inBlock = false

	return nil
}

// VisitPartial implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitPartial(node *PartialStatement) interface{} {
	// v.indent()
	v.str("{{> PARTIAL:")

	v.original = true
	node.Name.Accept(v)
	v.original = false

	if len(node.Params) > 0 {
		v.str(" ")
		node.Params[0].Accept(v)
	}

	// hash
	if node.Hash != nil {
		v.str(" ")
		node.Hash.Accept(v)
	}

	v.str(" }}")
	return nil
}

// VisitContent implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitContent(node *ContentStatement) interface{} {
	v.str(node.Original)
	return nil
}

// VisitComment implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitComment(node *CommentStatement) interface{} {
	v.line("{{! '" + node.Value + "' }}")

	return nil
}

// Expressions

// VisitExpression implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitExpression(node *Expression) interface{} {
	if v.inBlock {
		// v.str("{{#")
		// v.indent()
	}

	// path
	node.Path.Accept(v)

	// params
	for _, n := range node.Params {
		v.str(" ")
		n.Accept(v)
	}

	// hash
	if node.Hash != nil {
		v.str(" ")
		node.Hash.Accept(v)
	}

	if v.inBlock {
		// v.str("}}")
		// v.nl()
	}

	return nil
}

// VisitSubExpression implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitSubExpression(node *SubExpression) interface{} {
	node.Expression.Accept(v)

	return nil
}

// VisitPath implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitPath(node *PathExpression) interface{} {
	v.str(node.Original)
	return nil
}

// Literals

// VisitString implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitString(node *StringLiteral) interface{} {
	v.str(node.Value)
	return nil
}

// VisitBoolean implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitBoolean(node *BooleanLiteral) interface{} {
	v.str(node.Original)

	return nil
}

// VisitNumber implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitNumber(node *NumberLiteral) interface{} {
	v.str(node.Original)
	return nil
}

// Miscellaneous

// VisitHash implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitHash(node *Hash) interface{} {
	v.str("HASH{")

	for i, p := range node.Pairs {
		if i > 0 {
			v.str(", ")
		}
		p.Accept(v)
	}

	v.str("}")

	return nil
}

// VisitHashPair implements corresponding Visitor interface method
func (v *printOriginalVistor) VisitHashPair(node *HashPair) interface{} {
	v.str(node.Key + "=")
	node.Val.Accept(v)

	return nil
}
