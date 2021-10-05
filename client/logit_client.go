package main

import (
	"fmt"
	"github.com/chzyer/readline"
	"github.com/wbrown/novelai-research-tool/context"
	"log"
	"os"
	"strings"
	"unicode"
)

const colorGrey = "\033[38;5;240m"
const colorDkGrey = "\033[38;5;235m"
const colorWhite = "\033[0m"
const colorInput = "\033[30;47m"
const clearScreen = "'\033[2J'"

func cls() {
	print(clearScreen)
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

func printContextOutput(ctx context.SimpleContext, output string) {
	fmt.Println(colorDkGrey + ctx.Memory + "\n" + colorGrey + ctx.Context +
		colorWhite + output + colorDkGrey + "\n" + ctx.AuthorsNote + colorWhite)
}

func start() {
	ctx := context.NewSimpleContext()
	ctx.LastContext = ctx.Context
	cls()
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
	var splittext string
	var strlen int
	var array_context []string
	var array_output []string

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		switch line {
		case "BACK":
			if len(array_context) > 0 {
				array_context = array_context[:len(array_context)-1]
				array_output = array_output[:len(array_output)-1]
				ctx.Context = array_context[len(array_context)]
				output = array_output[len(array_output)]
			}
			printContextOutput(ctx, output)
			continue
		case "SAVE":
			ctx.SaveContext("prompt.txt")
			continue
		case "EDIT":
			defer rl.Close()
			readline.New(colorInput + ">" + colorWhite + output)

			fmt.Println(colorDkGrey + ctx.Memory + "\n" + colorGrey + ctx.Context +
				"\n" + colorDkGrey + ctx.AuthorsNote + colorWhite)

			datax := output
			data2 := []byte(datax)
			rl.WriteStdin(data2)

			ctx.Context = ctx.LastContext
			fmt.Println(colorGrey + ctx.Context)
			continue
		case "NEXT":
			ctx.Parameters.NextWord = true
			ctx.SaveContext("prompt.txt")
		}

		if ctx.Parameters.NextWord == false {
			ctx.Context = ctx.Context + line
		}
		splittext = ctx.Memory + "\n" + ctx.Context
		splitamt := float64(len(splittext)) * 0.025
		strlen = len(splittext) - 16 - int(splitamt)
		fulltext = splittext[:strlen] + ctx.AuthorsNote + splittext[strlen:]

		writeText("lastinput.txt", fulltext)
		for {
			tokens := ctx.Encoder.Encode(&fulltext)
			if uint(len(*tokens)) > ctx.MaxTokens {
				*tokens = (*tokens)[:2048]
				break

				ctx.Context = strings.Join(
					strings.Split(ctx.Context, "\n")[1:], "\n")
			} else {
				break
			}
		}
		resp := ctx.API.GenerateWithParams(&fulltext,
			ctx.Parameters)
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
		array_output = append(array_context, output)
		fmt.Println(colorDkGrey + ctx.Memory + "\n" +
			colorGrey + ctx.Context +
			colorWhite + output + colorDkGrey + "\n" +
			ctx.AuthorsNote + colorWhite)

		if ctx.Parameters.NextWord == false {
			ctx.LastContext = ctx.Context
			ctx.Context = ctx.Context + output
		}
	}
}

func main() {
	start()
}
