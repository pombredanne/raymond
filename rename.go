package raymond

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/komand/raymond/ast"
)

// renameVistor will go through a template and rename the variables come from steps.
type renameVisitor struct {
	// used for info on panic
	tpl       *Template
	curNode   ast.Node
	variables map[string]string

	blockStack []*ast.BlockStatement
}

func newRenameVisitor(tpl *Template, variables map[string]string) *renameVisitor {
	return &renameVisitor{
		tpl:       tpl,
		variables: variables,
	}
}

// at sets current node
func (v *renameVisitor) at(node ast.Node) {
	v.curNode = node
}

func (v *renameVisitor) VisitProgram(node *ast.Program) interface{} {
	v.at(node)

	for _, n := range node.Body {
		if err, _ := n.Accept(v).(error); err != nil {
			return err
		}
	}

	return nil
}

func (v *renameVisitor) VisitMustache(node *ast.MustacheStatement) interface{} {
	v.at(node)
	// evaluate expression
	return node.Expression.Accept(v)
}

// // statements
func (v *renameVisitor) VisitBlock(node *ast.BlockStatement) interface{} {
	v.at(node)

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

func (v *renameVisitor) inBlock() bool {
	return len(v.blockStack) > 0
}

func (v *renameVisitor) VisitPartial(node *ast.PartialStatement) interface{} {
	v.at(node)
	return nil
}

func (v *renameVisitor) VisitContent(node *ast.ContentStatement) interface{} {
	v.at(node)
	return nil
}
func (v *renameVisitor) VisitComment(node *ast.CommentStatement) interface{} {
	v.at(node)
	return nil
}

// expressions
func (v *renameVisitor) VisitExpression(node *ast.Expression) interface{} {
	done := false

	if node.Params != nil {
		for _, p := range node.Params {
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

func (v *renameVisitor) VisitSubExpression(node *ast.SubExpression) interface{} {
	v.at(node)
	return node.Expression.Accept(v)
}

func escapeString(str string) string {
	parts := strings.Split(str, ".")
	for id, part := range parts {
		if strings.Contains(part, " ") && !strings.HasPrefix(part, "[") {
			parts[id] = fmt.Sprintf("[%s]", part)
		}
	}

	return strings.Join(parts, ".")
}

func (v *renameVisitor) VisitPath(node *ast.PathExpression) interface{} {

	// perform strict validation
	if len(node.Parts) > 0 {
		for key, newVal := range v.variables {
			escapedKey := escapeString(key)
			newVal = escapeString(newVal)
			if strings.HasPrefix(node.Original, key) {
				node.Original = strings.Replace(node.Original, key, newVal, 1)
				node.Parts[0] = strings.Replace(node.Parts[0], key, newVal, 1)
			} else if strings.HasPrefix(node.Original, escapedKey) {
				node.Original = strings.Replace(node.Original, escapedKey, newVal, 1)
				node.Parts[0] = strings.Replace(node.Parts[0], escapedKey, newVal, 1)
			}

		}
	}

	return nil
}

// literals
func (v *renameVisitor) VisitString(node *ast.StringLiteral) interface{} {
	v.at(node)

	return nil
}
func (v *renameVisitor) VisitBoolean(node *ast.BooleanLiteral) interface{} {
	v.at(node)

	return nil
}
func (v *renameVisitor) VisitNumber(node *ast.NumberLiteral) interface{} {
	v.at(node)

	return nil
}

// miscellaneous
func (v *renameVisitor) VisitHash(node *ast.Hash) interface{} {
	v.at(node)

	for _, pair := range node.Pairs {
		if err := pair.Accept(v); err != nil {
			return err
		}
	}

	return nil
}

func (v *renameVisitor) VisitHashPair(node *ast.HashPair) interface{} {
	v.at(node)

	return node.Val.Accept(v)
}

// findHelper finds given helper
func (v *renameVisitor) findHelper(name string) reflect.Value {
	// check template helpers
	if h := v.tpl.findHelper(name); h != zero {
		return h
	}

	// check global helpers
	return findHelper(name)
}
