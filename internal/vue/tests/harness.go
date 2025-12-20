package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func ptrTo[T any](v T) *T {
	return &v
}

func withVueNodeModules(t *testing.T, content string) string {
	_, filename, _, _ := runtime.Caller(1)
	dirname := filepath.Dir(filename)
	var extraFilesBuilder strings.Builder
	extraFilesBuilder.WriteString("// @golarExtraFiles: ")

	err := filepath.Walk(filepath.Join(dirname, "node_modules"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".d.ts") || strings.HasSuffix(path, ".d.mts") || strings.HasSuffix(path, ".d.cts") || filepath.Base(path) == "package.json") {
			p, err := filepath.Rel(dirname, path)
			if err != nil {
				return err
			}
			virtualPath := filepath.Join("/", p)

			// https://en.wikipedia.org/wiki/Delimiter#Control_characters
			extraFilesBuilder.WriteString(path)
			extraFilesBuilder.WriteByte('\x1e')
			extraFilesBuilder.WriteString(virtualPath)
			extraFilesBuilder.WriteByte('\x1f')
		}
		return nil
	})
	assert.NilError(t, err)
	extraFilesBuilder.WriteByte('\n')

	return extraFilesBuilder.String() + content
}
