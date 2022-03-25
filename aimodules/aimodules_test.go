package aimodules

import "testing"

const modulePath = "/Users/wbrown/go/src/github.com/wbrown/novelai-research-tool/tests/OccultSage’s Genroku Era Module.module"

func TestAIModuleFromFile(t *testing.T) {
	AIModuleFromFile(modulePath)
}
