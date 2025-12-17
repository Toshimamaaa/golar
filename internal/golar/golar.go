package golar

import (
	"strings"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/internal/vue/codegen"
	"github.com/auvred/golar/internal/vue/parser"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/diagnosticwriter"
	"github.com/microsoft/typescript-go/shim/golarext"
	"github.com/microsoft/typescript-go/shim/parser"
)

type compilerHostProxy struct {
	compiler.CompilerHost
}

type languageData struct {
	sourceText string
	mapper     *mapping.Mapper
}

func (h *compilerHostProxy) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if strings.HasSuffix(opts.FileName, ".vue") {
		sourceText, ok := h.CompilerHost.FS().ReadFile(opts.FileName)
		if !ok {
			return nil
		}
		return parseFile(opts, sourceText, core.GetScriptKindFromFileName(opts.FileName))
	}
	return h.CompilerHost.GetSourceFile(opts)
}

func wrapCompilerHost(host compiler.CompilerHost) compiler.CompilerHost {
	return &compilerHostProxy{host}
}

type diagnosticProxy struct {
	*ast.Diagnostic
	cachedSourceLoc core.TextRange
	hasSource       bool
}

func newDiagnosticProxy(base *ast.Diagnostic) *diagnosticProxy {
	return &diagnosticProxy{
		Diagnostic:      base,
		cachedSourceLoc: core.NewTextRange(-1, -1),
	}
}

func (d *diagnosticProxy) sourceLoc() core.TextRange {
	if d.cachedSourceLoc.Pos() == -1 {
		if d.Diagnostic.Code() >= 1_000_000 {
			d.cachedSourceLoc = d.Diagnostic.Loc()
			d.hasSource = true
			return d.cachedSourceLoc
		}
		file := d.Diagnostic.File()
		if file != nil && file.GolarLanguageData != nil {
			langData := file.GolarLanguageData.(languageData)
			for _, sourceLoc := range langData.mapper.ToSourceRange(d.Diagnostic.Pos(), d.Diagnostic.End(), true) {
				d.cachedSourceLoc = core.NewTextRange(sourceLoc.MappedStart, sourceLoc.MappedEnd)
				d.hasSource = true
				return d.cachedSourceLoc
			}
		}
		d.cachedSourceLoc = d.Diagnostic.Loc()
	}
	return d.cachedSourceLoc
}

func (d *diagnosticProxy) RelatedInformation() []diagnosticwriter.Diagnostic {
	related := d.Diagnostic.RelatedInformation()
	result := []diagnosticwriter.Diagnostic{}
	for _, r := range related {
		relProxy := newDiagnosticProxy(r)
		if r.Code() >= 1_000_000 {
			result = append(result, relProxy)
			continue
		}
		relProxy.sourceLoc()
		if relProxy.hasSource {
			result = append(result, relProxy)
		}
	}
	return result
}

func (d *diagnosticProxy) MessageChain() []diagnosticwriter.Diagnostic {
	chain := d.Diagnostic.MessageChain()
	result := []diagnosticwriter.Diagnostic{}
	for _, r := range chain {
		relProxy := newDiagnosticProxy(r)
		if r.Code() >= 1_000_000 {
			result = append(result, relProxy)
			continue
		}
		relProxy.sourceLoc()
		if relProxy.hasSource {
			result = append(result, relProxy)
		}
	}
	return result
}

type fileProxy struct {
	*ast.SourceFile
}

func (f *fileProxy) Text() string {
	return f.SourceFile.GolarLanguageData.(languageData).sourceText
}

func (d *diagnosticProxy) File() diagnosticwriter.FileLike {
	if file := d.Diagnostic.File(); file != nil {
		if file.GolarLanguageData == nil {
			return file
		}
		return &fileProxy{file}
	}
	return nil
}

func (d *diagnosticProxy) Loc() core.TextRange {
	return d.sourceLoc()
}

func (d *diagnosticProxy) Len() int {
	return d.sourceLoc().Len()
}

func (d *diagnosticProxy) Pos() int {
	return d.sourceLoc().Pos()
}

func (d *diagnosticProxy) End() int {
	return d.sourceLoc().End()
}

func wrapASTDiagnostic(diagnostic *ast.Diagnostic) diagnosticwriter.Diagnostic {
	return newDiagnosticProxy(diagnostic)
}

func parseFile(opts ast.SourceFileParseOptions, sourceText string, scriptKind core.ScriptKind) *ast.SourceFile {
	if !strings.HasSuffix(opts.FileName, ".vue") {
		return parser.ParseSourceFile(opts, sourceText, scriptKind)
	}
	ast := vue_parser.Parse(sourceText)
	serviceText, mappings, codegenDiagnostics := vue_codegen.Codegen(sourceText, ast)
	file := parser.ParseSourceFile(opts, serviceText, scriptKind)
	for _, d := range codegenDiagnostics {
		d.SetFile(file)
		for _, r := range d.RelatedInformation() {
			r.SetFile(file)
		}
	}
	file.SetDiagnostics(append(file.Diagnostics(), codegenDiagnostics...))
	file.GolarLanguageData = languageData{
		sourceText: sourceText,
		mapper:     mapping.NewMapper(mappings),
	}

	return file
}

func adjustDiagnostic(file *ast.SourceFile, diagnostic *ast.Diagnostic) *ast.Diagnostic {
	if file.GolarLanguageData == nil || diagnostic.Code() >= 1_000_000 {
		return diagnostic
	}
	langData := file.GolarLanguageData.(languageData)
	for _, sourceRange := range langData.mapper.ToSourceRange(diagnostic.Pos(), diagnostic.End(), true) {
		diagnostic.SetLocation(core.NewTextRange(sourceRange.MappedStart, sourceRange.MappedEnd))
		break
	}

	return diagnostic
}

func positionToService(file *ast.SourceFile, pos int) int {
	if file.GolarLanguageData == nil {
		return pos
	}

	langData := file.GolarLanguageData.(languageData)
	for _, serviceLoc := range langData.mapper.ToServiceLocation(pos) {
		return serviceLoc.Offset
	}
	return pos
}

var GolarExtCallbacks = &golarext.GolarCallbacks{
	AdjustDiagnostic:  adjustDiagnostic,
	PositionToService: positionToService,
	WrapCompilerHost:  wrapCompilerHost,
	WrapASTDiagnostic: wrapASTDiagnostic,
	ParseSourceFile:   parseFile,
}
