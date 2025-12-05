package main

import (
	"fmt"
	"os"

	"github.com/microsoft/typescript-go/shim/bundled"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/microsoft/typescript-go/shim/vfs/cachedvfs"
	"github.com/microsoft/typescript-go/shim/vfs/osvfs"
)

func getProgram(cwd string) (*compiler.Program, error) {

	fs := bundled.WrapFS(cachedvfs.From(osvfs.FS()))
	var configFileName string
	var tsconfig string // TODO
	if tsconfig == "" {
		configFileName = tspath.ResolvePath(cwd, "tsconfig.json")
		if !fs.FileExists(configFileName) {
			return nil, fmt.Errorf("couldn't find tsconfig.json")
			// fs = utils.NewOverlayVFS(fs, map[string]string{
			// 	configFileName: "{}",
			// })
		}
	} else {
		configFileName = tspath.ResolvePath(cwd, tsconfig)
		if !fs.FileExists(configFileName) {
			return nil, fmt.Errorf("error: tsconfig %q doesn't exist", tsconfig)
		}
	}

	cwd = tspath.GetDirectoryPath(configFileName)

	host := CreateCompilerHost(cwd, fs)

	// comparePathOptions := tspath.ComparePathsOptions{
	// 	cwd:          host.GetCurrentDirectory(),
	// 	UseCaseSensitiveFileNames: host.FS().UseCaseSensitiveFileNames(),
	// }

	var singleThreaded bool // TODO
	program, err := CreateProgram(singleThreaded, fs, cwd, configFileName, &host)
	if err != nil {
		return nil, fmt.Errorf("error creating TS program: %v", err)
	}

	return program, nil
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting current directory: %v\n", err)
		os.Exit(1)
	}
	cwd = tspath.NormalizePath(cwd)

	program, err := getProgram(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating program: %v\n", err)
		os.Exit(1)
	}

	index := program.GetSourceFileByPath(tspath.Path(tspath.ResolvePath(cwd, "index.ts")))
	comp := program.GetSourceFileByPath(tspath.Path(tspath.ResolvePath(cwd, "Comp.vue")))
	println(index, comp)
}
