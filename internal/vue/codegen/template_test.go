package vue_codegen

import (
	"strconv"
	"testing"

	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/parser"
	"github.com/microsoft/typescript-go/shim/core"

	// "gotest.tools/v3/assert"
	"github.com/google/go-cmp/cmp"
)

func TestExpressionMapper(t *testing.T) {
	t.Run("non-binding position", func (t *testing.T) {
		cases := []struct{
			sourceText string
			serviceText string
		}{
			{
				"hello",
				" __VLS_Ctx.hello",
			},
			{
				"hello.world",
				" __VLS_Ctx.hello.world",
			},
			{
				"hello[world]",
				" __VLS_Ctx.hello[ __VLS_Ctx.world]",
			},
			{
				"() => { const foo: SomeType = bar }",
				"() => { const foo: SomeType = __VLS_Ctx. bar }",
			},
			{
				"() => { return foo }",
				"() => { return __VLS_Ctx. foo }",
			},
			{
				"{ a: a }",
				"{ a: __VLS_Ctx. a }",
			},
			{
				"{ a:  }",
				"{ a: __VLS_Ctx.  }",
			},
			{
				"{ a }",
				"{a: __VLS_Ctx. a }",
			},
			{
				"{ [a]: a }",
				"{ [ __VLS_Ctx.a]: __VLS_Ctx. a }",
			},
			{
				"() => { class foo {} }",
				"() => { class foo {} }",
			},
			{
				"() => { interface foo { hello: world } }",
				"() => { interface foo { hello: world } }",
			},
			{
				"() => { foo: while (1) break foo }",
				"() => { foo: while (1) break foo }",
			},
			{
				"() => { foo: while (1) continue foo }",
				"() => { foo: while (1) continue foo }",
			},
			{
				"() => { type foo = bar }",
				"() => { type foo = bar }",
			},
			{
				"() => { enum foo { a = hello, b }}",
				"() => { enum foo { a = __VLS_Ctx. hello, b }}",
			},
			{
				"() => { const [, value] = foo }",
				"() => { const [, value] = __VLS_Ctx. foo }",
			},
			{
				"() => { const { [foo]: bar = baz } = qux }",
				"() => { const { [ __VLS_Ctx.foo]: bar = __VLS_Ctx. baz } = __VLS_Ctx. qux }",
			},
			// {
			// 	"() => { function foo(): bar {} }",
			// 	"() => { function foo(): bar {} }",
			// },
			// {
			// 	"() => { function foo(): typeof bar {} }",
			// 	"() => { function foo(): typeof __VLS_Ctx. bar {} }",
			// },
			// {
			// 	"() => { function foo<T extends bar>() {} }",
			// 	"() => { function foo<T extends bar>() {} }",
			// },
			// {
			// 	"() => { function foo<T extends typeof bar.baz[typeof qux]>() {} }",
			// 	"() => { function foo<T extends typeof __VLS_Ctx. bar.baz[typeof __VLS_Ctx. qux]>() {} }",
			// },
			{
				"{ ...foo }",
				"{ ... __VLS_Ctx.foo }",
			},
		}

		for i, c := range cases {
			t.Run(strconv.Itoa(i), func (t *testing.T) {
				base := newCodegenCtx(nil, c.sourceText)
				ctx := newTemplateCodegenCtx(&base)

				tsAst := vue_parser.ParseTsAst("(" + c.sourceText + ")")
				expr := vue_ast.NewSimpleExpressionNode(tsAst, core.NewTextRange(0, len(c.sourceText)), 1, 1)
				ctx.mapExpressionInNonBindingPosition(expr)

				diff := cmp.Diff(c.serviceText, ctx.serviceText.String())
				if diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	t.Run("binding position", func (t *testing.T) {
		cases := []struct{
			sourceText string
			serviceText string
		}{
			{"hello", "hello"},
		}

		for i, c := range cases {
			t.Run(strconv.Itoa(i), func (t *testing.T) {
				base := newCodegenCtx(nil, c.sourceText)
				ctx := newTemplateCodegenCtx(&base)

				tsAst := vue_parser.ParseTsAst("(" + c.sourceText + ")=>{}")
				expr := vue_ast.NewSimpleExpressionNode(tsAst, core.NewTextRange(0, len(c.sourceText)), 1, 5)
				ctx.mapExpressionInBindingPosition(expr)

				diff := cmp.Diff(c.serviceText, ctx.serviceText.String())
				if diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}
