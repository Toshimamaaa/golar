package vue_codegen

import (
	vue_ast "github.com/auvred/golar/internal/vue/ast"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

// TODO: <script src="">

type scriptCodegenCtx struct {
	*codegenCtx
	scriptSetupEl *vue_ast.ElementNode
	scriptEl      *vue_ast.ElementNode
}

func generateScript(base *codegenCtx, scriptSetupEl *vue_ast.ElementNode, scriptEl *vue_ast.ElementNode) {
	c := scriptCodegenCtx{
		codegenCtx:    base,
		scriptSetupEl: scriptSetupEl,
		scriptEl:      scriptEl,
	}

	c.serviceText.WriteString("import { defineComponent as __VLS_DefineComponent } from 'vue'\n")

	var selfType string
	if c.scriptEl != nil {
		if len(c.scriptEl.Children) != 1 {
			panic("TODO: len of <script> children != 1")
		}

		innerStart := c.scriptEl.InnerLoc.Pos()
		text := c.scriptEl.Children[0].AsText()

		mapStart := text.Loc.Pos()
		hasExportDefault := false

		for _, statement := range c.scriptEl.Ast.Statements.Nodes {
			if !ast.IsExportAssignment(statement) {
				continue
			}

			hasExportDefault = true
			export := statement.AsExportAssignment()
			c.mapText(mapStart, innerStart+export.Expression.Pos())
			c.serviceText.WriteString(" {} as unknown as typeof __VLS_Export\n")
			if c.scriptSetupEl == nil {
				c.serviceText.WriteString("const __VLS_Export = ")
				selfType = "__VLS_Export"
			} else {
				c.serviceText.WriteString("const __VLS_Self = ")
				selfType = "__VLS_Self"
			}
			mapStart = innerStart + export.Expression.Pos()

			break
		}

		c.mapText(mapStart, text.Loc.End())
		c.serviceText.WriteString("\n\n")

		if !hasExportDefault {
			c.serviceText.WriteString("const __VLS_Export = __VLS_DefineComponent({})\nexport default __VLS_Export\n")
		}

		// TODO: options wrapper - wrap export default |defineComponent(|{}|)|
	}

	// TODO: generic support
	if c.scriptSetupEl != nil {
		if len(c.scriptSetupEl.Children) != 1 {
			panic("TODO: len of <script setup> children != 1")
		}

		text := c.scriptSetupEl.Children[0].AsText()

		if c.scriptEl != nil {
			c.serviceText.WriteString("const __VLS_Export = await (async () => {\n")
		} else {
			// TODO
			c.serviceText.WriteString("const __VLS_Export = __VLS_DefineComponent({})\n")
		}
		innerStart := c.scriptSetupEl.InnerLoc.Pos()

		lastMappedPos := text.Loc.Pos()

		bindingRanges := []core.TextRange{}
		importRanges := []core.TextRange{}
		for _, statement := range c.scriptSetupEl.Ast.Statements.Nodes {
			switch statement.Kind {
			case ast.KindVariableStatement:
				for _, decl := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
					name := decl.AsVariableDeclaration().Name()
					var visitor ast.Visitor
					visitor = func(n *ast.Node) bool {
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
			case ast.KindImportDeclaration:
				if c.scriptEl != nil {
					importRanges = append(importRanges, core.NewTextRange(innerStart+statement.Loc.Pos(), innerStart+statement.Loc.End()))
					if lastMappedPos != statement.Pos() {
						c.mapText(lastMappedPos, innerStart+statement.Pos())
					}
					lastMappedPos = innerStart + statement.End()
				}
				importClause := statement.AsImportDeclaration().ImportClause
				if importClause != nil {
					if importClause.Name() != nil {
						bindingRanges = append(bindingRanges, importClause.Name().Loc)
					}

					namedBindings := importClause.AsImportClause().NamedBindings
					if namedBindings != nil {
						if ast.IsNamespaceImport(namedBindings) {
							bindingRanges = append(bindingRanges, namedBindings.Name().Loc)
						} else {
							for _, element := range namedBindings.Elements() {
								bindingRanges = append(bindingRanges, element.Name().Loc)
							}
						}
					}
				}
			}
		}
		c.mapText(lastMappedPos, text.Loc.End())
		c.serviceText.WriteByte('\n')

		if len(bindingRanges) > 0 {
			c.serviceText.WriteString("type __VLS_SetupExposed = {\n")
			// TODO: proxy refs
			for _, binding := range bindingRanges {
				c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
				c.serviceText.WriteString(": typeof ")
				c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
				c.serviceText.WriteRune('\n')
			}
			c.serviceText.WriteString("}\n")
		}

		c.serviceText.WriteString("const __VLS_Ctx = {\n")
		if len(bindingRanges) > 0 {
			c.serviceText.WriteString("...{} as unknown as __VLS_SetupExposed,\n")
		}
		if selfType != "" {
			c.serviceText.WriteString("...{} as unknown as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(", new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as unknown as import('vue').ComponentPublicInstance,\n")
		}
		c.serviceText.WriteString("}\n")

		if c.scriptEl != nil {
			c.serviceText.WriteString("\n})()\n")
			for _, loc := range importRanges {
				c.mapText(loc.Pos(), loc.End())
				c.serviceText.WriteString("\n")
			}
		}

		if c.scriptEl == nil {
			c.serviceText.WriteString("export default {} as unknown as Awaited<typeof __VLS_Export>\n")
		}
	}
}
