package tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestQuickInfo(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	const /*1*/foo = 'hello'
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", `const foo: "hello"`, "")
}
