package vue_codegen

import (
	"github.com/auvred/golar/internal/collections"
	vue_ast "github.com/auvred/golar/internal/vue/ast"
	vue_diagnostics "github.com/auvred/golar/internal/vue/diagnostics"
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

type conditionalChain uint8

const (
	conditionalChainNone conditionalChain = iota
	conditionalChainValid
	conditionalChainBroken
)

func (c *templateCodegenCtx) visit(el *vue_ast.ElementNode) {
	condChain := conditionalChainNone
	for _, child := range el.Children {
		switch child.Kind {
		case vue_ast.KindElement:
			elem := child.AsElement()

			var conditionalDirective *vue_ast.DirectiveNode
			var seenProps collections.Set[string]
			hasSeenConditionalDirective := false

			for _, p := range elem.Props {
				if p.Kind != vue_ast.KindDirective {
					attr := p.AsAttribute()
					if seenProps.Has(attr.Name) {
						c.reportDiagnostic(attr.NameLoc, vue_diagnostics.Elements_cannot_have_multiple_X_0_with_the_same_name, "attributes")
					} else {
						seenProps.Add(attr.Name)
					}
					continue
				}
				dir := p.AsDirective()
				if seenProps.Has(dir.RawName) {
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Elements_cannot_have_multiple_X_0_with_the_same_name, "directives")
					continue
				} else {
					seenProps.Add(dir.RawName)
				}
				switch dir.Name {
				case "if":
					if hasSeenConditionalDirective {
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
						break
					}
					condChain = conditionalChainValid
					conditionalDirective = dir
				case "else-if":
					if hasSeenConditionalDirective {
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
						break
					}
					hasSeenConditionalDirective = true
					switch condChain {
					case conditionalChainNone:
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else-if")
						condChain = conditionalChainBroken
					case conditionalChainValid:
						conditionalDirective = dir
					}
				case "else":
					if hasSeenConditionalDirective {
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
						break
					}
					hasSeenConditionalDirective = true
					switch condChain {
					case conditionalChainNone:
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else")
					case conditionalChainValid:
						condChain = conditionalChainNone
						conditionalDirective = dir
					}
				}
			}
			if conditionalDirective != nil {
				switch conditionalDirective.Name {
				case "else-if":
					c.serviceText.WriteString("else ")
					fallthrough
				case "if":
					c.serviceText.WriteString("if (")
					if conditionalDirective.Expression != nil && conditionalDirective.Expression.Ast != nil {
						c.genExpressionWithPrefixedIdentifiers(conditionalDirective.Expression)
					} else {
						c.reportDiagnostic(conditionalDirective.Loc, vue_diagnostics.X_0_is_missing_expression, conditionalDirective.RawName)
						c.serviceText.WriteString("true")
					}
					c.serviceText.WriteString(") {\n")
					c.visit(elem)
					c.serviceText.WriteString("}\n")
				case "else":
					c.serviceText.WriteString("else {\n")
					c.visit(elem)
					c.serviceText.WriteString("}\n")
				}
			} else {
				if !hasSeenConditionalDirective {
					condChain = conditionalChainNone
				}
				c.visit(elem)
			}
		case vue_ast.KindInterpolation:
			interpolation := child.AsInterpolation()
			c.serviceText.WriteString(";( ")
			c.genExpressionWithPrefixedIdentifiers(interpolation.Content)
			c.serviceText.WriteString(" )\n")
		}
	}
}

func (c *templateCodegenCtx) genExpressionWithPrefixedIdentifiers(expr *vue_ast.SimpleExpressionNode) {
	innerStart := expr.Loc.Pos() - expr.PrefixLen
	lastEnd := expr.Loc.Pos()
	var visitor ast.Visitor
	visitor = func(node *ast.Node) bool {
		switch node.Kind {
		case ast.KindIdentifier:
			if c.shouldPrefixIdentifier(node) {
				c.mapText(lastEnd, innerStart+node.Pos())
				c.serviceText.WriteString(" __VLS_Ctx.")
				c.mapText(innerStart+node.Pos(), innerStart+node.End())
				lastEnd = innerStart + node.End()
			}
			return false
		case ast.KindShorthandPropertyAssignment:
			name := node.Name()
			if c.shouldPrefixIdentifier(name) {
				c.mapText(lastEnd, innerStart+node.Pos())
				c.serviceText.WriteString(name.Text())
				c.serviceText.WriteString(": __VLS_Ctx.")
				c.mapText(innerStart+node.Pos(), innerStart+node.End())
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
	visitor(expr.Ast.AsNode())
	c.mapText(lastEnd, expr.Loc.End())
}
