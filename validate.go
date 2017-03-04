package raymond

import (
	"fmt"
	"reflect"

	"github.com/komand/raymond/ast"
)

// validateVistor will go through a template and validate the variables come from steps.
type validateVisitor struct {
	// used for info on panic
	tpl       *Template
	curNode   ast.Node
	variables map[string]struct{}

	blockStack []*ast.BlockStatement
}

func newValidateVisitor(tpl *Template, variables map[string]struct{}) *validateVisitor {
	return &validateVisitor{
		tpl:        tpl,
		variables:  variables,
		blockStack: make([]*ast.BlockStatement, 0),
	}
}

// at sets current node
func (v *validateVisitor) at(node ast.Node) {
	v.curNode = node
}

func (v *validateVisitor) VisitProgram(node *ast.Program) interface{} {
	v.at(node)

	for _, n := range node.Body {
		if err, _ := (n.Accept(v)).(error); err != nil {
			return err
		}
	}

	return nil
}

func (v *validateVisitor) VisitMustache(node *ast.MustacheStatement) interface{} {
	v.at(node)
	// fmt.Println("XXX MS", *node)

	// evaluate expression
	return node.Expression.Accept(v)
}

// // statements
func (v *validateVisitor) VisitBlock(node *ast.BlockStatement) interface{} {
	v.at(node)

	// fmt.Printf("XXX BLOCK %#v\n", *node)

	v.blockStack = append(v.blockStack, node)
	// evaluate expression
	err := node.Expression.Accept(v)
	if err != nil {
		return err
	}
	if node.Program != nil {
		err := node.Program.Accept(v)
		if err != nil {
			return err
		}
	}
	if node.Inverse != nil {
		err := node.Inverse.Accept(v)
		if err != nil {
			return err
		}
	}

	v.blockStack = v.blockStack[:len(v.blockStack)-1]
	return err
}

func (v *validateVisitor) inBlock() bool {
	return len(v.blockStack) > 0
}

func (v *validateVisitor) VisitPartial(node *ast.PartialStatement) interface{} {
	v.at(node)
	return nil
}

func (v *validateVisitor) VisitContent(node *ast.ContentStatement) interface{} {
	v.at(node)
	return nil
}
func (v *validateVisitor) VisitComment(node *ast.CommentStatement) interface{} {
	v.at(node)
	return nil
}

// expressions
func (v *validateVisitor) VisitExpression(node *ast.Expression) interface{} {
	done := false

	// fmt.Printf("XXX EXPRESSION %#v\nIN BLOCk=%s\n", *node, v.inBlock())

	if node.Params != nil {
		for _, p := range node.Params {
			// fmt.Println("XXX PARAM", p)
			err := p.Accept(v)
			if err != nil {
				return err
			}
		}
	}
	// helper call
	if helperName := node.HelperName(); helperName != "" {
		if helper := v.findHelper(helperName); helper != zero {
			// it's a valid helper
			done = true
		}
	}

	if !done {
		// literals are skipped
		if _, ok := node.LiteralStr(); ok {
			return nil
		}
	}

	if !done {
		// field path
		if path := node.FieldPath(); path != nil {
			if err := v.VisitPath(path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *validateVisitor) VisitSubExpression(node *ast.SubExpression) interface{} {
	v.at(node)
	return node.Expression.Accept(v)
}

func (v *validateVisitor) VisitPath(node *ast.PathExpression) interface{} {

	if v.inBlock() {
		// looser validation requirements - let's just ignore for now.
	} else {
		// perform strict validation
		if len(node.Parts) > 0 {
			found := false
			for val, _ := range v.variables {
				escapedVal := fmt.Sprintf("[%s]", val)
				if val == node.Parts[0] || escapedVal == node.Parts[0] {
					found = true
				}
			}

			if !found {
				return fmt.Errorf("Invalid variable reference %s (not in %#v)", node.Original, v.variables)
			}
		}
	}

	return nil
}

// literals
func (v *validateVisitor) VisitString(node *ast.StringLiteral) interface{} {
	v.at(node)

	return nil
}
func (v *validateVisitor) VisitBoolean(node *ast.BooleanLiteral) interface{} {
	v.at(node)

	return nil
}
func (v *validateVisitor) VisitNumber(node *ast.NumberLiteral) interface{} {
	v.at(node)

	return nil
}

// miscellaneous
func (v *validateVisitor) VisitHash(node *ast.Hash) interface{} {
	v.at(node)

	for _, pair := range node.Pairs {
		if err := pair.Accept(v); err != nil {
			return err
		}
	}

	return nil
}

func (v *validateVisitor) VisitHashPair(node *ast.HashPair) interface{} {
	v.at(node)

	return node.Val.Accept(v)
}

// findHelper finds given helper
func (v *validateVisitor) findHelper(name string) reflect.Value {
	// check template helpers
	if h := v.tpl.findHelper(name); h != zero {
		return h
	}

	// check global helpers
	return findHelper(name)
}
