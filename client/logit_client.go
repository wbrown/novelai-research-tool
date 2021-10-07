package main

import (
	"fmt"
	"github.com/chzyer/readline"
	"github.com/inancgumus/screen"
	"github.com/wbrown/novelai-research-tool/context"
	"log"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

const colorGrey = "\033[38;5;240m"
const colorDkGrey = "\033[38;5;235m"
const colorWhite = "\033[0m"
const colorInput = "\033[30;47m"

func fixcmd() {
	cmd := exec.Command("cmd")
	cmd.Stdout = os.Stdout
	cmd.Run()
	cls()
}

func cls() {
	screen.Clear()
	screen.MoveTopLeft()
}

func pause() {
	fmt.Scanln()
}

func writeText(path string, text string) {
	if f, err := os.Create(path); err != nil {
		println("\n\n\n\nError saving file.")
		log.Fatal(err)
	} else if _, err = f.WriteString(text); err != nil {
		println("\n\n\n\nError saving file.")
		log.Fatal(err)
	}
}

func refresh(context_ string, output_ string, ctx context.SimpleContext) {
	cls()
	fmt.Println(colorDkGrey + ctx.Memory + "\n" + colorGrey + context_ + colorWhite + output_ +
		"\n" + colorDkGrey + ctx.AuthorsNote + colorWhite)
}

func start() {
	cls()
	ctx := context.NewSimpleContext()
	ctx.LastContext = ctx.Context
	fmt.Println(colorDkGrey + ctx.Memory + "\n" +
		colorGrey + ctx.Context + "\n" +
		colorDkGrey + ctx.AuthorsNote + colorWhite)

	rl, err := readline.New(colorInput + ">" + colorWhite)
	if err != nil {
		println("\n\n\n\nError making newline.")
		pause()
		panic(err)
	}
	defer rl.Close()

	var output string
	var fulltext string
	var array_context []string
	var array_output []string
	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}

		ctx.Parameters.NextWord = false
		switch line {
		case "BACK":
			if len(array_context) > 0 {
				ctx.Context = array_context[len(array_context)-1]
				output = array_context[len(array_output)-1]
				array_context = array_context[:len(array_context)-1]
				array_output = array_output[:len(array_output)-1]
			}
			refresh(ctx.Context, "", ctx)
			continue
		case "SAVE":
			ctx.SaveContext("prompt.txt")
			refresh(ctx.Context, "", ctx)
			continue
		case "EDIT":
			defer rl.Close()
			readline.New(colorInput + ">" + colorWhite + output)
			refresh(ctx.Context, "", ctx)
			datax := output
			data2 := []byte(datax)
			rl.WriteStdin(data2)

			ctx.Context = ctx.LastContext
			fmt.Println(colorGrey + ctx.Context)
			continue
		}

		if line == "NEXT" {
			line = ""
			ctx.Parameters.NextWord = true
		}

		if line == "RETRY" {
			line = ""
			output = ""
			ctx.Context = ctx.LastContext
			refresh(ctx.Context, "", ctx)
		}

		line = strings.TrimSpace(line)
		ctx.Context = ctx.Context + line
		fulltext = ctx.Memory + "\n" + ctx.Context
		if len(ctx.AuthorsNote) > 0 {
			splittext := strings.Split(fulltext, "\n")
			insertPos := len(splittext)-2
			if insertPos < 0 {
				insertPos = 0
			}
			rest := append([]string{ctx.AuthorsNote}, splittext[insertPos:]...)
			splittext = append(splittext[:insertPos], rest...)
			fulltext = strings.Join(splittext, "\n")
		}
		fulltext = strings.TrimRight(fulltext, "\n")
		writeText("lastinput.txt", fulltext)
		for {
			tokens := ctx.Encoder.Encode(&fulltext)
			if uint(len(*tokens)) > ctx.MaxTokens {
				fulltext = strings.Join(
					strings.Split(ctx.Context, "\n")[1:], "\n")
			} else {
				break
			}
		}
		resp := ctx.API.GenerateWithParams(&fulltext, ctx.Parameters)
		output = resp.Response

		var eos_pos int
		var eos_exit bool
		eos_pos = len(output) - 2
		for eos_pos >= 0 {
			curChar := output[eos_pos]
			eos_exit = false
			if unicode.IsPunct(rune(curChar)) && eos_exit == false && ctx.FullReturn == false && eos_pos < len(output) {
				output = output[:eos_pos+1]
				eos_exit = true
			}
			eos_pos = eos_pos - 1
		}

		if ctx.Parameters.NextWord == true {
			fmt.Println(colorWhite + "\nPRESS ENTER TO CONTINUE...\n")
			pause()
		}

		array_context = append(array_context, ctx.Context)
		array_output = append(array_output, output)

		refresh(ctx.Context, output, ctx)

		if ctx.Parameters.NextWord == false {
			ctx.LastContext = ctx.Context
			ctx.Context = ctx.Context + output
		}

	}
}

func main() {
	fixcmd()
	start()
}
