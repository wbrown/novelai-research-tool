package main

import (
	"encoding/json"
	"fmt"
	"github.com/chzyer/readline"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"log"
	"strings"
	"unicode"
    "os"
    "os/exec"
)

//var f embed.FS
const colorGrey = "\033[38;5;240m"
const colorDkGrey = "\033[38;5;235m"
const colorWhite = "\033[0m"
const colorInput = "\033[30;47m"


type Adventure struct {
	Parameters novelai_api.NaiGenerateParams
	Context    string
	Memory     string
	FullReturn bool
	AuthorsNote string
	LastContext string
	API        novelai_api.NovelAiAPI
	Encoder    gpt_bpe.GPTEncoder
	MaxTokens  uint
}

func NewAdventure() (adventure Adventure) {
	parametersFile, _ := os.ReadFile("params.json")
	var parameters novelai_api.NaiGenerateParams
	if err := json.Unmarshal(parametersFile, &parameters); err != nil {
	println("Error in JSON.")
	fmt.Scanln()
		log.Fatal(err)
		
	}
	
	contextBytes, _ := os.ReadFile("prompt.txt")
	adventure.Context = string(contextBytes)
	adventure.LastContext = string(contextBytes)
	adventure.Parameters = parameters
	adventure.API = novelai_api.NewNovelAiAPI()
	adventure.Encoder = gpt_bpe.NewEncoder()
	adventure.FullReturn = parameters.ReturnFullText
	adventure.MaxTokens = 2048 - *parameters.MaxLength
	adventure.Memory = parameters.LogitMemory
	adventure.AuthorsNote = parameters.LogitAuthors
	return adventure
}

func (adventure Adventure) start() {
	adventure.Context = adventure.Context
	adventure.LastContext = adventure.Context
	rl, err := readline.New(colorInput+">"+colorWhite)
	if err != nil {
		println("\n\n\n\nError making newline.")
	fmt.Scanln()
		panic(err)
	}
	defer rl.Close()

	var output string
	var send bool
	var fulltext string
	var splittext string
	var strlen int
	var array_context [65536]string
	var array_output [65536]string
	var array_pos int 
	
	send = true
	for {

		line, err := rl.Readline()
		
		if err != nil { // io.EOF
			break
		}
		
		
		if line == "BACK"{
		send = false
		
		if array_pos >0{
		array_pos = array_pos -1
		
		adventure.Context = array_context[array_pos]
		output = array_output[array_pos]
		
	cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested 
        cmd.Stdout = os.Stdout
        cmd.Run()
	fmt.Println(colorDkGrey+adventure.Memory+"\n"+colorGrey+adventure.Context+colorWhite+output+colorDkGrey+"\n"+adventure.AuthorsNote+colorWhite)
		}
		
		}
		
		if line == "SAVE"{
		send = false
		
				f, err := os.Create("prompt.txt")
		_, err2 := f.WriteString(adventure.Context)
		
	 if err != nil {
	println("\n\n\n\nError saving file.")
	fmt.Scanln()
        log.Fatal(err)
		fmt.Scanln()
    }
	
	    if err2 != nil {
	println("\n\n\n\nError saving file.")
	fmt.Scanln()
        log.Fatal(err)
		fmt.Scanln()
    }	
		}
		
		adventure.Parameters.NextWord = false
		if line == "NEXT"{
		adventure.Parameters.NextWord = true
		fmt.Println("\nEXPECTED NEXT TOKENS...\n")
		
		}
		
	if line == "EDIT"{
	send = false
				
				
	defer rl.Close()
		readline.New(colorInput+">"+colorWhite+output)
				
		fmt.Println(colorDkGrey+adventure.Memory+"\n"+colorGrey+adventure.Context+"\n"+colorDkGrey+adventure.AuthorsNote+colorWhite)
				
		datax := output;
        data2 := []byte(datax);
        rl.WriteStdin(data2);
			
		adventure.Context = adventure.LastContext
		fmt.Println(colorGrey+adventure.Context)				
				
				}
		
		if line == "RETRY"{
        send = true
		adventure.Context = adventure.LastContext
		
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested 
        cmd.Stdout = os.Stdout
        cmd.Run()
		fmt.Println(colorDkGrey+adventure.Memory+"\n"+colorGrey+adventure.Context+"\n"+colorDkGrey+adventure.AuthorsNote+colorWhite)

        line = ""
		}

		if send == true{
		
		if adventure.Parameters.NextWord == false{
		adventure.Context = adventure.Context + line}
		splittext = adventure.Memory+"\n"+adventure.Context
		splitamt := float64(len(splittext)) *0.025
		strlen = len(splittext)-16-int(splitamt)
		fulltext = splittext[:strlen] +adventure.AuthorsNote + splittext[strlen:]
		
		f, err := os.Create("lastinput.txt")
		_, err2 := f.WriteString(fulltext)
		
	 if err != nil {
	println("\n\n\n\nError saving file.")
	fmt.Scanln()
        log.Fatal(err)
		fmt.Scanln()
    }
	
	    if err2 != nil {
	println("\n\n\n\nError saving file.")
	fmt.Scanln()
        log.Fatal(err)
		fmt.Scanln()
    }	
		
		for {
			tokens := adventure.Encoder.Encode(&fulltext)
			if uint(len(*tokens)) > adventure.MaxTokens{
			 (*tokens) = (*tokens)[:2048]
			 break
			 
				adventure.Context = strings.Join(
					strings.Split(adventure.Context, "\n")[1:], "\n")
			} else {
				break
			}
		}
		resp := adventure.API.GenerateWithParams(&fulltext,
			adventure.Parameters)
		output = resp.Response
		
		
		var eos_pos int
		var eos_exit bool
		eos_pos = len(output)-2
		for eos_pos >=0{
		
		curChar := output[eos_pos]
		eos_exit = false
			if unicode.IsPunct(rune(curChar)) && eos_exit == false && adventure.FullReturn == false && eos_pos <len(output){
				output = output[:eos_pos+1]
				eos_exit = true
			}
		eos_pos = eos_pos-1
		}
		
		if adventure.Parameters.NextWord == true{
		fmt.Println(colorWhite+"\nPRESS ENTER TO CONTINUE...\n")
        fmt.Scanln()		
		}
		
        cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested 
        cmd.Stdout = os.Stdout
        cmd.Run()
		
		array_context[array_pos] = adventure.Context
		array_output[array_pos] = output
		array_pos = array_pos +1
		

		
		fmt.Println(colorDkGrey+adventure.Memory+"\n"+colorGrey+adventure.Context+colorWhite+output+colorDkGrey+"\n"+adventure.AuthorsNote+colorWhite)
		
		if adventure.Parameters.NextWord == false{
		adventure.LastContext = adventure.Context
		adventure.Context = adventure.Context + output
		}
		}else{

		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested 
        cmd.Stdout = os.Stdout
        cmd.Run()
		fmt.Println(colorDkGrey+adventure.Memory+"\n"+colorGrey+adventure.Context+colorWhite+output+colorDkGrey+"\n"+adventure.AuthorsNote+colorWhite)
		 send = true
	defer rl.Close()
		readline.New(colorInput+">"+colorWhite)
			if err != nil {
	println("\n\n\n\nError creating newline.")
	fmt.Scanln()
		panic(err)
		fmt.Scanln()
	}

				 }
	}
}

func main() {	

	adventure := NewAdventure()

	cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested 
    cmd.Stdout = os.Stdout
    cmd.Run()
	fmt.Println(colorDkGrey+adventure.Memory+"\n"+colorGrey+adventure.Context+"\n"+colorDkGrey+adventure.AuthorsNote+colorWhite)
	adventure.start()
}
