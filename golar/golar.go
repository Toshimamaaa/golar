package golar

import (
	"strings"
	"sync"

	"github.com/auvred/golar/vue/codegen"
	"github.com/auvred/golar/vue/parser"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/microsoft/typescript-go/shim/golarext"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/diagnosticwriter"
	"github.com/microsoft/typescript-go/shim/parser"
)

type compilerHostProxy struct {
	compiler.CompilerHost
}

type languageData struct {
	sourceText string
	mappings   []vue_codegen.Mapping
}

func (h *compilerHostProxy) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if strings.HasSuffix(opts.FileName, ".vue") {
		sourceText, ok := h.CompilerHost.FS().ReadFile(opts.FileName)
		if !ok {
			return nil
		}
		return ParseFile(opts, sourceText, core.GetScriptKindFromFileName(opts.FileName))
	}
	return h.CompilerHost.GetSourceFile(opts)
}

func WrapCompilerHost(host compiler.CompilerHost) compiler.CompilerHost {
	return &compilerHostProxy{host}
}

type diagnosticProxy struct {
	*ast.Diagnostic
}

func (d *diagnosticProxy) RelatedInformation() []diagnosticwriter.Diagnostic {
	related := d.Diagnostic.RelatedInformation()
	result := make([]diagnosticwriter.Diagnostic, len(related))
	for i, r := range related {
		result[i] = &diagnosticProxy{r}
	}
	return result
}

type fileProxy struct {
	orig          *ast.SourceFile
	ecmaLineMapMu sync.RWMutex
	ecmaLineMap   []core.TextPos
}

func (f *fileProxy) FileName() string {
	return f.orig.FileName()
}

func (f *fileProxy) Text() string {
	return f.orig.GolarLanguageData.(languageData).sourceText
}

func (f *fileProxy) ECMALineMap() []core.TextPos {
	f.ecmaLineMapMu.RLock()
	lineMap := f.ecmaLineMap
	f.ecmaLineMapMu.RUnlock()
	if lineMap == nil {
		f.ecmaLineMapMu.Lock()
		defer f.ecmaLineMapMu.Unlock()
		lineMap = f.ecmaLineMap
		if lineMap == nil {
			lineMap = core.ComputeECMALineStarts(f.Text())
			f.ecmaLineMap = lineMap
		}
	}
	return lineMap
}

func (d *diagnosticProxy) File() diagnosticwriter.FileLike {
	if file := d.Diagnostic.File(); file != nil {
		if file.GolarLanguageData == nil {
			return file
		}
		return &fileProxy{
			orig: file,
		}
	}
	return nil
}

func (d *diagnosticProxy) MessageChain() []diagnosticwriter.Diagnostic {
	chain := d.Diagnostic.MessageChain()
	result := make([]diagnosticwriter.Diagnostic, len(chain))
	for i, c := range chain {
		result[i] = &diagnosticProxy{c}
	}
	return result
}

func (d *diagnosticProxy) Pos() int {
	file := d.Diagnostic.File()
	if file == nil || file.GolarLanguageData == nil {
		return d.Diagnostic.Pos()
	}
	servicePos := d.Diagnostic.Pos()
	for _, m := range file.GolarLanguageData.(languageData).mappings {
		if m.ServiceOffset <= servicePos && servicePos < m.ServiceOffset + m.Length {
			return servicePos - m.ServiceOffset + m.SourceOffset
		}
	}
	return servicePos
}

func (d *diagnosticProxy) End() int {
	file := d.Diagnostic.File()
	if file == nil || file.GolarLanguageData == nil {
		return d.Diagnostic.Pos()
	}
	servicePos := d.Diagnostic.End()
	for _, m := range file.GolarLanguageData.(languageData).mappings {
		if m.ServiceOffset <= servicePos && servicePos < m.ServiceOffset + m.Length {
			return servicePos - m.ServiceOffset + m.SourceOffset
		}
	}
	return servicePos
}

func WrapASTDiagnostic(diagnostic *ast.Diagnostic) diagnosticwriter.Diagnostic {
	return &diagnosticProxy{diagnostic}
}

func ParseFile(opts ast.SourceFileParseOptions, sourceText string, scriptKind core.ScriptKind) *ast.SourceFile {
	if !strings.HasSuffix(opts.FileName, ".vue") {
		return parser.ParseSourceFile(opts, sourceText, scriptKind)
	}
	ast := vue_parser.Parse(sourceText)
	serviceText, mappings := vue_codegen.Codegen(sourceText, ast)
	file := parser.ParseSourceFile(opts, serviceText, scriptKind)
	file.GolarLanguageData = languageData{
		sourceText: sourceText,
		mappings:   mappings,
	}

	return file
}

func adjustDiagnostic(file *ast.SourceFile, diagnostic *ast.Diagnostic) *ast.Diagnostic {
	if file.GolarLanguageData == nil {
		return diagnostic
	}

	servicePos := diagnostic.Pos()
	for _, m := range file.GolarLanguageData.(languageData).mappings {
		if m.ServiceOffset <= servicePos && servicePos < m.ServiceOffset + m.Length {
			pos := servicePos - m.ServiceOffset + m.SourceOffset
			diagnostic.SetLocation(core.NewTextRange(pos, pos + diagnostic.Len()))
			break
		}
	}

	return diagnostic
}

func positionToService(file *ast.SourceFile, pos int) int {
	if file.GolarLanguageData == nil {
		return pos
	}

	for _, m := range file.GolarLanguageData.(languageData).mappings {
		if m.SourceOffset <= pos && pos < m.SourceOffset + m.Length {
			return pos - m.SourceOffset + m.ServiceOffset
		}
	}
	return pos
}

func positionToSource(file *ast.SourceFile, pos int) int {
	if file.GolarLanguageData == nil {
		return pos
	}

	for _, m := range file.GolarLanguageData.(languageData).mappings {
		if m.ServiceOffset <= pos && pos < m.ServiceOffset + m.Length {
			return pos - m.ServiceOffset + m.SourceOffset
		}
	}

	return pos
}

var GolarExtCallbacks = &golarext.GolarCallbacks{
	AdjustDiagnostic: adjustDiagnostic,
	PositionToService: positionToService,
	PositionToSource: positionToSource,
}
