package vue_codegen

import (
	"github.com/auvred/golar/internal/collections"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
)

type templateCodegenCtx struct {
	*codegenCtx
	scopes []collections.Set[string]
}

func newTemplateCodegenCtx(base *codegenCtx) templateCodegenCtx {
	return templateCodegenCtx{
		codegenCtx: base,
	}
}

func generateTemplate(base *codegenCtx, el *vue_ast.ElementNode) {
	c := newTemplateCodegenCtx(base)
	if el != nil {
		c.visit(el)
	}
}

func (c *templateCodegenCtx) enterScope() {
	c.scopes = append(c.scopes, collections.Set[string]{})
}
func (c *templateCodegenCtx) exitScope() {
	if len(c.scopes) > 0 {
		c.scopes = c.scopes[:len(c.scopes)-1]
	}
}
func (c *templateCodegenCtx) declareScopeVar(name string) {
	if len(c.scopes) > 0 {
		c.scopes[len(c.scopes)-1].Add(name)
	}
}

func (c *templateCodegenCtx) shouldPrefixIdentifier(identifier *ast.Node) bool {
	name := identifier.Text()

	for location := identifier; location != nil; location = location.Parent {
		locals := location.Locals()
		if _, ok := locals[name]; ok {
			return false
		}
	}

	for _, scope := range c.scopes {
		if scope.Has(name) {
			return false
		}
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
			var forDirective *vue_ast.ForParseResult
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
				case "for":
					forDirective = dir.ForParseResult
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
						c.mapExpressionInNonBindingPosition(conditionalDirective.Expression)
					} else {
						c.reportDiagnostic(conditionalDirective.Loc, vue_diagnostics.X_0_is_missing_expression, conditionalDirective.RawName)
						c.serviceText.WriteString("true")
					}
					c.serviceText.WriteString(") {\n")
				case "else":
					c.serviceText.WriteString("else {\n")
				}
			} else if !hasSeenConditionalDirective {
				condChain = conditionalChainNone
			}
			if forDirective != nil {
				c.enterScope()
				c.serviceText.WriteString("{\nconst [")
				if forDirective.Value != nil {
					c.mapExpressionInBindingPosition(forDirective.Value)
				}
				c.serviceText.WriteString(",")
				if forDirective.Key != nil {
					c.mapExpressionInBindingPosition(forDirective.Key)
				}
				c.serviceText.WriteString(",")
				if forDirective.Index != nil {
					c.mapExpressionInBindingPosition(forDirective.Index)
				}
				c.serviceText.WriteString("] = __VLS_vFor(")
				c.mapExpressionInNonBindingPosition(forDirective.Source)
				c.serviceText.WriteString(")\n")
			}
			c.visit(elem)
			if forDirective != nil {
				c.exitScope()
				c.serviceText.WriteString("}\n")
			}
			if conditionalDirective != nil {
				c.serviceText.WriteString("}\n")
			}
		case vue_ast.KindInterpolation:
			interpolation := child.AsInterpolation()
			c.serviceText.WriteString(";( ")
			c.mapExpressionInNonBindingPosition(interpolation.Content)
			c.serviceText.WriteString(" )\n")
		}
	}
}

type expressionMapper struct {
	*templateCodegenCtx
	expr          *vue_ast.SimpleExpressionNode
	innerStart    int
	lastMappedPos int
	typeOnly      bool
}

func newExpressionMapper(c *templateCodegenCtx, expr *vue_ast.SimpleExpressionNode) expressionMapper {
	return expressionMapper{
		templateCodegenCtx: c,
		expr:               expr,
		innerStart:         expr.Loc.Pos() - expr.PrefixLen,
		lastMappedPos:      expr.Loc.Pos(),
	}
}

func (m *expressionMapper) mapTextToNodePos(pos int) {
	pos += m.innerStart
	m.mapText(m.lastMappedPos, pos)
	m.lastMappedPos = pos
}

func (c *templateCodegenCtx) mapExpressionInNonBindingPosition(expr *vue_ast.SimpleExpressionNode) {
	m := newExpressionMapper(c, expr)
	if len(expr.Ast.Statements.Nodes) > 0 {
		firstStmt := expr.Ast.Statements.Nodes[0]
		// TODO: report non-binding cases
		if ast.IsExpressionStatement(firstStmt) {
			expr := firstStmt.AsExpressionStatement().Expression
			if ast.IsParenthesizedExpression(expr) {
				m.mapInNonBindingPosition(expr.AsParenthesizedExpression().Expression)
			}
		}
	}
	m.mapTextToNodePos(expr.Ast.End() - expr.SuffixLen)
}
func (c *templateCodegenCtx) mapExpressionInBindingPosition(expr *vue_ast.SimpleExpressionNode) {
	m := newExpressionMapper(c, expr)
	if len(expr.Ast.Statements.Nodes) > 0 {
		firstStmt := expr.Ast.Statements.Nodes[0]
		// TODO: report non-binding cases
		if ast.IsExpressionStatement(firstStmt) {
			expr := firstStmt.AsExpressionStatement().Expression
			if ast.IsArrowFunction(expr) {
				fn := expr.AsArrowFunction()
				if len(fn.Parameters.Nodes) == 1 && ast.IsParameter(fn.Parameters.Nodes[0]) {
					m.mapInBindingPosition(fn.Parameters.Nodes[0].AsParameterDeclaration().Name())
				}
			}
		}
	}
	m.mapTextToNodePos(expr.Ast.End() - expr.SuffixLen)
}

func (m *expressionMapper) mapInBindingPosition(node *ast.BindingName) bool {
	switch node.Kind {
	case ast.KindIdentifier:
		m.declareScopeVar(node.AsIdentifier().Text)
	case ast.KindArrayBindingPattern, ast.KindObjectBindingPattern:
		for _, elem := range node.AsBindingPattern().Elements.Nodes {
			bindingElem := elem.AsBindingElement()
			if bindingElem.PropertyName != nil && m.mapInNonBindingPosition(bindingElem.PropertyName) {
				return true
			}
			if bindingElem.Name() != nil && m.mapInBindingPosition(bindingElem.Name()) {
				return true
			}
			if bindingElem.Initializer != nil && m.mapInNonBindingPosition(bindingElem.Initializer) {
				return true
			}
		}
	}
	return false
}

func visit(v ast.Visitor, node *ast.Node) bool {
	if node != nil {
		return v(node)
	}
	return false
}

func (m *expressionMapper) typeOnlyVisit(v ast.Visitor, node *ast.Node) bool {
	before := m.typeOnly
	m.typeOnly = true
	res := visit(v, node)
	m.typeOnly = before
	return res
}

func (m *expressionMapper) mapInNonBindingPositionIfNotIdentifier(node *ast.Node) bool {
	return !ast.IsIdentifier(node) && m.mapInNonBindingPosition(node)
}

// TODO: more robust support for types, etc.
func (m *expressionMapper) mapInNonBindingPosition(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindIdentifier:
		if m.shouldPrefixIdentifier(node) {
			m.mapTextToNodePos(node.Pos())
			m.serviceText.WriteString(" __VLS_Ctx.")
			m.mapTextToNodePos(node.End())
		}
		return false
	case ast.KindShorthandPropertyAssignment:
		name := node.Name()
		if m.shouldPrefixIdentifier(name) {
			m.mapTextToNodePos(node.Pos())
			m.serviceText.WriteString(name.Text())
			m.serviceText.WriteString(": __VLS_Ctx.")
			m.mapTextToNodePos(node.End())
		}
		return false
	case ast.KindPropertyAccessExpression:
		n := node.AsPropertyAccessExpression()
		return visit(m.mapInNonBindingPosition, n.Expression) || visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name())
	case ast.KindEnumMember:
		n := node.AsEnumMember()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindPropertySignature:
		n := node.AsPropertySignatureDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindPropertyAssignment:
		n := node.AsPropertyAssignment()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || visit(m.mapInNonBindingPosition, n.Initializer)
	// TODO: maybe we can track locals in codegen scope instead of relying on binder?
	// TODO: class decl, function decl, enum, etc.
	case ast.KindVariableDeclaration:
		decl := node.AsVariableDeclaration()
		return visit(m.mapInBindingPosition, decl.Name()) || visit(m.mapInBindingPosition, decl.Type) || visit(m.mapInNonBindingPosition, decl.Initializer)
	case ast.KindBreakStatement, ast.KindContinueStatement, ast.KindLabeledStatement:
		return false
	}
	if ast.IsTypeNode(node) {
		return false
	}

	return node.ForEachChild(m.mapInNonBindingPosition)
}
