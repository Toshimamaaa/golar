module github.com/auvred/golar

go 1.25.0

replace (
	github.com/microsoft/typescript-go/shim/ast => ./shim/ast
	github.com/microsoft/typescript-go/shim/binder => ./shim/binder
	github.com/microsoft/typescript-go/shim/bundled => ./shim/bundled
	github.com/microsoft/typescript-go/shim/checker => ./shim/checker
	github.com/microsoft/typescript-go/shim/compiler => ./shim/compiler
	github.com/microsoft/typescript-go/shim/core => ./shim/core
	github.com/microsoft/typescript-go/shim/diagnostics => ./shim/diagnostics
	github.com/microsoft/typescript-go/shim/diagnosticwriter => ./shim/diagnosticwriter
	github.com/microsoft/typescript-go/shim/golarext => ./shim/golarext
	github.com/microsoft/typescript-go/shim/lsp/lsproto => ./shim/lsp/lsproto
	github.com/microsoft/typescript-go/shim/parser => ./shim/parser
	github.com/microsoft/typescript-go/shim/project => ./shim/project
	github.com/microsoft/typescript-go/shim/scanner => ./shim/scanner
	github.com/microsoft/typescript-go/shim/tsoptions => ./shim/tsoptions
	github.com/microsoft/typescript-go/shim/tspath => ./shim/tspath
	github.com/microsoft/typescript-go/shim/vfs => ./shim/vfs
	github.com/microsoft/typescript-go/shim/vfs/cachedvfs => ./shim/vfs/cachedvfs
	github.com/microsoft/typescript-go/shim/vfs/osvfs => ./shim/vfs/osvfs
)

require (
	github.com/microsoft/typescript-go/shim/ast v0.0.0
	github.com/microsoft/typescript-go/shim/binder v0.0.0
	github.com/microsoft/typescript-go/shim/bundled v0.0.0
	github.com/microsoft/typescript-go/shim/checker v0.0.0
	github.com/microsoft/typescript-go/shim/compiler v0.0.0
	github.com/microsoft/typescript-go/shim/core v0.0.0
	github.com/microsoft/typescript-go/shim/lsp/lsproto v0.0.0
	github.com/microsoft/typescript-go/shim/project v0.0.0
	github.com/microsoft/typescript-go/shim/scanner v0.0.0
	github.com/microsoft/typescript-go/shim/diagnostics v0.0.0
	github.com/microsoft/typescript-go/shim/diagnosticwriter v0.0.0
	github.com/microsoft/typescript-go/shim/golarext v0.0.0
	github.com/microsoft/typescript-go/shim/parser v0.0.0
	github.com/microsoft/typescript-go/shim/tsoptions v0.0.0
	github.com/microsoft/typescript-go/shim/tspath v0.0.0
	github.com/microsoft/typescript-go/shim/vfs v0.0.0
	github.com/microsoft/typescript-go/shim/vfs/cachedvfs v0.0.0
	github.com/microsoft/typescript-go/shim/vfs/osvfs v0.0.0
	golang.org/x/text v0.31.0
	golang.org/x/tools v0.38.0
)

require (
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/go-json-experiment/json v0.0.0-20251027170946-4849db3c2f7e // indirect
	github.com/microsoft/typescript-go v0.0.0-20251204215308-2ae410164f65 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)
