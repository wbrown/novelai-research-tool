package main

import (
	"fmt"
	"github.com/wbrown/novelai-research-tool/aimodules"
	"os"
)

func main() {
	module := aimodules.AIModuleFromFile(os.Args[1])
	fmt.Printf("NAME: %v\nID: %v\nDESCRIPTION: %v\nSTEPS: %v\n",
		module.Name, module.ToPrefix(), module.Description, module.Steps)
}