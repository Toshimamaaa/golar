package vue_codegen

import (
	"strings"

	"github.com/auvred/golar/internal/vue/ast"
	vue_diagnostics "github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/diagnostics"
)

type Mapping struct {
	SourceOffset int
	ServiceOffset int
	Length int
}

func Codegen(sourceText string, root *vue_ast.RootNode) (string, []Mapping, []*ast.Diagnostic) {
	ctx := newCodegenCtx(root, sourceText)

	var scriptSetupCtx *codegenCtx
	var templateCtx *codegenCtx

	for _, child := range root.Children {
		if child.Type != vue_ast.NodeTypeELEMENT {
			continue
		}

		el := child.AsElement()
		if el.Tag == "script" {
			for _, prop := range el.Props {
				if prop.Type == vue_ast.NodeTypeATTRIBUTE {
					attr := prop.AsAttribute()
					if attr.Name == "setup" {
						if scriptSetupCtx != nil {
							ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_script_setup_element)
							break
						}
						ctx := newCodegenCtx(root, sourceText)
						scriptSetupCtx = &ctx
						generateScriptSetup(scriptSetupCtx, el)
					}
				}
			}
		}

		if el.Tag == "template" {
			if templateCtx != nil {
				ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_template_element)
				continue
			}
			ctx := newCodegenCtx(root, sourceText)
			templateCtx = &ctx
			generateTemplate(templateCtx, el)
		}
	}

	// https://github.com/volarjs/volar.js/discussions/188
	lineStart := 0
	for {
		idx := strings.IndexByte(sourceText[lineStart:], '\n')
		if idx == -1 {
			for range len(sourceText) - lineStart {
				ctx.serviceText.WriteByte(' ')
			}
			break
		}
		idx += lineStart
		for range idx - lineStart {
			ctx.serviceText.WriteByte(' ')
		}
		ctx.serviceText.WriteByte('\n')
		lineStart = idx + 1
	}

	if scriptSetupCtx != nil {
		newMappingsStart := len(ctx.mappings)
		ctx.mappings = append(ctx.mappings, scriptSetupCtx.mappings...)
		for i := newMappingsStart; i < len(ctx.mappings); i++ {
			ctx.mappings[i].ServiceOffset += ctx.serviceText.Len()
		}
		ctx.serviceText.Write([]byte(scriptSetupCtx.serviceText.String()))
	}

	if templateCtx != nil {
		newMappingsStart := len(ctx.mappings)
		ctx.mappings = append(ctx.mappings, templateCtx.mappings...)
		for i := newMappingsStart; i < len(ctx.mappings); i++ {
			ctx.mappings[i].ServiceOffset += ctx.serviceText.Len()
		}
		ctx.serviceText.Write([]byte(templateCtx.serviceText.String()))
	}

	return ctx.serviceText.String(), ctx.mappings, ctx.diagnostics
}

type codegenCtx struct {
	ast *vue_ast.RootNode
	sourceText string
	serviceText strings.Builder
	mappings []Mapping
	diagnostics []*ast.Diagnostic
}

func newCodegenCtx(root *vue_ast.RootNode, sourceText string) codegenCtx {
	return codegenCtx{
		ast: root,
		sourceText: sourceText,
		serviceText: strings.Builder{},
		mappings: []Mapping{},
		diagnostics: []*ast.Diagnostic{},
	}
}

func (c *codegenCtx) reportDiagnostic(loc core.TextRange, message *diagnostics.Message) {
	c.diagnostics = append(c.diagnostics, ast.NewDiagnostic(nil, loc, message))
}

func (c *codegenCtx) mapText(from, to int) {
	serviceOffset := c.serviceText.Len()
	c.serviceText.WriteString(c.sourceText[from:to])
	c.mappings = append(c.mappings, Mapping{
		SourceOffset: from,
		ServiceOffset: serviceOffset,
		Length: to - from,
	})
}

func  generateScriptSetup(c *codegenCtx, el *vue_ast.ElementNode) {
	if len(el.Children) != 1 {
		panic("TODO: len of <script> children != 1")
	}

	text := el.Children[0].AsText()

	c.serviceText.WriteString("// hello from codegen\n\n")
	c.mapText(text.Loc.Pos(), text.Loc.End())
	c.serviceText.WriteString("\n\n")

	bindingRanges := []core.TextRange{}
	for _, statement := range el.Ast.Statements.Nodes {
		switch statement.Kind {
		case ast.KindVariableStatement:
			for _, decl := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
				name := decl.AsVariableDeclaration().Name()
				var visitor ast.Visitor
				visitor = func (n *ast.Node) bool {
					if ast.IsIdentifier(n) {
						bindingRanges = append(bindingRanges, n.Loc)
					}
					return n.ForEachChild(visitor)
				}
				visitor(name)
			}
		case ast.KindFunctionDeclaration, ast.KindClassDeclaration, ast.KindEnumDeclaration:
			if name := statement.Name(); name != nil {
				bindingRanges = append(bindingRanges, name.Loc)
			}
		}
	}

	innerStart := el.InnerLoc.Pos()

	if len(bindingRanges) > 0 {
		c.serviceText.WriteString("type __VLS_SetupExposed = {\n")
		// TODO: proxy refs
		for _, binding := range bindingRanges {
			c.serviceText.WriteString(c.sourceText[innerStart + binding.Pos():innerStart + binding.End()])
			c.serviceText.WriteString(": typeof ")
			c.serviceText.WriteString(c.sourceText[innerStart + binding.Pos():innerStart + binding.End()])
			c.serviceText.WriteRune('\n')
		}
		c.serviceText.WriteString("}\n")
	}

	c.serviceText.WriteString("const __VLS_Ctx = {\n")
	if len(bindingRanges) > 0 {
		c.serviceText.WriteString("...{} as __VLS_SetupExposed,\n")
	}
	c.serviceText.WriteString("}\n")

	c.serviceText.WriteString("export default {} as import('vue').Component\n")
}

type templateCodegenCtx struct {
	*codegenCtx
}
func generateTemplate(base *codegenCtx, el *vue_ast.ElementNode) {
	c := templateCodegenCtx{
		codegenCtx: base,
	}
	c.visit(el)
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

