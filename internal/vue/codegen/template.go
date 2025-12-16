package vue_codegen

import (
	vue_ast "github.com/auvred/golar/internal/vue/ast"
	"github.com/microsoft/typescript-go/shim/ast"
)

type templateCodegenCtx struct {
	*codegenCtx
	scopes []map[string]struct{}
}
func generateTemplate(base *codegenCtx, el *vue_ast.ElementNode) {
	c := templateCodegenCtx{
		codegenCtx: base,
		scopes: []map[string]struct{}{},
	}
	if el != nil {
		c.visit(el)
	}
}

func (c *templateCodegenCtx) shouldPrefixIdentifier(identifier *ast.Node) bool {
	name := identifier.Text()
	location := identifier
	for location != nil {
		locals := location.Locals()
		if _, ok := locals[name]; ok {
			return false
		}
		location = location.Parent
	}
	return true
}

func (c *templateCodegenCtx) visit(el *vue_ast.ElementNode) {
	for _, child := range el.Children {
		switch child.Type {
		case vue_ast.NodeTypeELEMENT:
			c.visit(child.AsElement())
		case vue_ast.NodeTypeINTERPOLATION:
			interpolation := child.AsInterpolation()
			c.serviceText.WriteString(";( ")
			innerStart := interpolation.Loc.Pos() + 2
			lastEnd := innerStart
			var visitor ast.Visitor
			visitor = func (node *ast.Node) bool {
				switch node.Kind {
				case ast.KindIdentifier:
					if c.shouldPrefixIdentifier(node) {
						c.mapText(lastEnd, innerStart + node.Pos())
						c.serviceText.WriteString(" __VLS_Ctx.")
						c.mapText(innerStart + node.Pos(), innerStart + node.End())
						lastEnd = innerStart + node.End()
					}
					return false
				case ast.KindShorthandPropertyAssignment:
					name := node.Name()
					if c.shouldPrefixIdentifier(name) {
						c.mapText(lastEnd, innerStart + node.Pos())
						c.serviceText.WriteString(name.Text())
						c.serviceText.WriteString(": __VLS_Ctx.")
						c.mapText(innerStart + node.Pos(), innerStart + node.End())
						lastEnd = innerStart + node.End()
					}
					return false
				case ast.KindVariableDeclaration:
					decl := node.AsVariableDeclaration()
					if name := decl.Name(); name != nil && !ast.IsIdentifier(name) {
						if name.ForEachChild(visitor) {
							return true
						}
					}
					return (decl.Type != nil && decl.Type.ForEachChild(visitor)) || (decl.Initializer != nil && decl.Initializer.ForEachChild(visitor))
				case ast.KindArrayBindingPattern, ast.KindObjectBindingPattern:
					for _, elem := range node.AsBindingPattern().Elements.Nodes {
						if !ast.IsIdentifier(elem) && elem.ForEachChild(visitor) {
							return true
						}
					}
				}
				return node.ForEachChild(visitor)
			}
			visitor(interpolation.Content.Ast.AsNode())
			c.mapText(lastEnd, interpolation.Loc.End() - 2)
			c.serviceText.WriteString(" )\n")
		}
	}
}

