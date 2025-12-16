package vue_codegen

import (
	vue_ast "github.com/auvred/golar/internal/vue/ast"
	"github.com/microsoft/typescript-go/shim/ast"
)

type templateCodegenCtx struct {
	*codegenCtx
}
func generateTemplate(base *codegenCtx, el *vue_ast.ElementNode) {
	c := templateCodegenCtx{
		codegenCtx: base,
	}
	if el != nil {
		c.visit(el)
	}
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
				// TODO: skip in binding positions
				if ast.IsIdentifier(node) {
					c.mapText(lastEnd, innerStart + node.Pos())
					c.serviceText.WriteString("__VLS_Ctx.")
					c.mapText(innerStart + node.Pos(), innerStart + node.End())
					lastEnd = innerStart + node.End()
				}
				return node.ForEachChild(visitor)
			}
			visitor(interpolation.Content.Ast.AsNode())
			c.mapText(lastEnd, interpolation.Loc.End() - 2)
			c.serviceText.WriteString(" )\n")
		}
	}
}

