package gpt_bpe

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	lru "github.com/hashicorp/golang-lru"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
)

//go:embed encoder.json
//go:embed vocab.bpe

var f embed.FS

const BPE_LRU_SZ = 8192

type Token uint16
type Tokens []Token

type GPTEncoder struct {
	encoder    map[string]Token
	decoder    map[Token][]byte
	bpe_ranks  map[GPTPair]float64
	pattern    *regexp.Regexp
	byteToRune [256]rune
	runeToByte map[rune]byte
	cache      *lru.Cache
}

type GPTPair struct {
	left  string
	right string
}

type BGERank struct {
	rank   float64
	bigram GPTPair
}

type BGERanks []BGERank

func (bs BGERanks) Len() int {
	return len(bs)
}

func (bs BGERanks) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}

func (bs BGERanks) Less(i, j int) bool {
	return bs[i].rank < bs[j].rank
}

func NewEncoder() GPTEncoder {
	// Read encoder mappings and also generate reverse mappings.
	encoderFile, _ := f.ReadFile("encoder.json")
	encoderTokens := make(map[string]Token)
	json.Unmarshal(encoderFile, &encoderTokens)
	tokensEncoder := make(map[Token][]byte)
	for text, token := range encoderTokens {
		tokensEncoder[token] = []byte(text)
	}
	// Read vocabulary into bpe_ranks
	bpeRanks := make(map[GPTPair]float64)
	bpeMergesFile, _ := f.ReadFile("vocab.bpe")
	scanner := bufio.NewScanner(bytes.NewBuffer(bpeMergesFile))
	idx := uint16(0)
	firstLine := true
	for scanner.Scan() {
		if firstLine == true {
			firstLine = false
			continue
		}
		left_right := strings.SplitN(scanner.Text(), " ", 2)
		bpeRanks[GPTPair{left_right[0], left_right[1]}] = float64(idx)
		idx += 1
	}
	pat, err := regexp.Compile("'s|'t|'re|'ve|'m|'ll|'d| ?\\p{L}+| ?\\p{N}+| ?[^\\s\\p{L}\\p{N}]+|\\s+(\\S){0}|\\s+")
	if err != nil {
		log.Printf("gpt_bpe: Fatal error compiling regular expression: %v", err)
		os.Exit(1)
	}
	// Build the bytes to unicode tables.
	bytesUnicodeMap := make(map[byte]rune)
	unicodeBytes := make(map[rune]byte)
	for b := uint8('!'); b < uint8('~')+1; b++ {
		bytesUnicodeMap[b] = rune(b)
		unicodeBytes[rune(b)] = b
	}
	for b := uint8('¡'); b < uint8('¬')+1; b++ {
		bytesUnicodeMap[b] = rune(b)
		unicodeBytes[rune(b)] = b
	}
	for b := uint16('®'); b < uint16('ÿ')+1; b++ {
		bytesUnicodeMap[byte(b)] = rune(b)
		unicodeBytes[rune(b)] = byte(b)
	}
	uct := 0
	var bytesUnicode [256]rune
	for b := Token(0); b < 256; b++ {
		if _, ok := bytesUnicodeMap[uint8(b)]; !ok {
			bytesUnicodeMap[uint8(b)] = rune(256 + uct)
			unicodeBytes[rune(256+uct)] = uint8(b)
			uct += 1
		}
		bytesUnicode[b] = bytesUnicodeMap[uint8(b)]
	}
	cache, _ := lru.New(BPE_LRU_SZ)

	return GPTEncoder{
		encoderTokens,
		tokensEncoder,
		bpeRanks,
		pat,
		bytesUnicode,
		unicodeBytes,
		cache,
	}
}

// insertAt inserts v into s at index i and returns the new slice.
func insertAt(data []BGERank, i int, v BGERank) []BGERank {
	if i == len(data) {
		// Insert at end is the easy case.
		return append(data, v)
	}

	// Make space for the inserted element by shifting
	// values at the insertion index up one index. The call
	// to append does not allocate memory when cap(data) is
	// greater ​than len(data).
	data = append(data[:i+1], data[i:]...)

	// Insert the new element.
	data[i] = v

	// Return the updated slice.
	return data
}

func insertSortedNoDups(data BGERanks, v BGERank) BGERanks {
	i := sort.Search(len(data), func(i int) bool { return data[i].rank >= v.rank })
	if i < len(data) && data[i] == v {
		return data
	}
	return insertAt(data, i, v)
}

func getPairs(word []string) []GPTPair {
	pairsSet := make(map[GPTPair]bool, len(word))
	pairs := make([]GPTPair, len(word))
	begin := 1
	prev := word[0]
	ct := 0
	for idx := begin; idx < len(word); idx++ {
		present := word[idx]
		pair := GPTPair{prev, present}
		if _, ok := pairsSet[pair]; !ok {
			pairs[len(pairsSet)] = pair
			ct++
		}
		pairsSet[pair] = true
		prev = present
	}
	return pairs[0:ct]
}

func (encoder *GPTEncoder) getRankedPairs(word []string) BGERanks {
	rankedPairs := make(BGERanks, 0, len(word))
	begin := 1
	prev := word[0]
	for idx := begin; idx < len(word); idx++ {
		present := word[idx]
		pair := GPTPair{prev, present}
		bpe, ok := encoder.bpe_ranks[pair]
		if !ok {
			bpe = math.Inf(1)
		}
		rankedPairs = insertSortedNoDups(rankedPairs, BGERank{bpe, pair})
		prev = present
	}
	return rankedPairs
}

func (encoder *GPTEncoder) rankPairs(pairs []GPTPair) BGERanks {
	rankedPairs := make(BGERanks, 0)
	for idx := range pairs {
		bpe, ok := encoder.bpe_ranks[pairs[idx]]
		if !ok {
			bpe = math.Inf(1)
		}
		rankedPairs = insertSortedNoDups(rankedPairs, BGERank{bpe, pairs[idx]})
	}
	sort.Sort(rankedPairs)
	return rankedPairs
}

func (encoder *GPTEncoder) minPair(pairs []GPTPair) (retPair GPTPair) {
	rankedPairs := encoder.rankPairs(pairs)
	if len(rankedPairs) > 0 {
		retPair = rankedPairs[0].bigram
	}
	return retPair
}

func (encoder *GPTEncoder) toUnicode(text *string) string {
	textBytes := []byte(*text)
	outArr := make([]rune, len(*text))
	for idx := range textBytes {
		outArr[idx] = encoder.byteToRune[textBytes[idx]]
	}
	return string(outArr)
}

func pos(word []string, seek string, i int) int {
	for j, v := range word[i:] {
		if seek == v {
			return j + i
		}
	}
	return -1
}

func (encoder *GPTEncoder) encodeTokens(tokens *[]string) (encoded Tokens) {
	for idx := range *tokens {
		encoded = append(encoded, encoder.encoder[(*tokens)[idx]])
	}
	return encoded
}

func (encoder *GPTEncoder) toBPE(text string) []string {
	if lookup, ok := encoder.cache.Get(text); ok {
		return lookup.([]string)
	}
	word := strings.Split(text, "")
	rankedPairs := encoder.getRankedPairs(word)
	if len(rankedPairs) == 0 {
		return []string{text}
	}
	for {
		bigram := rankedPairs[0].bigram
		if _, ok := encoder.bpe_ranks[bigram]; !ok {
			break
		}
		first := bigram.left
		second := bigram.right
		newWord := make([]string, 0, len(word))
		for i := 0; i < len(word); {
			j := pos(word, first, i)
			if j == -1 {
				newWord = append(newWord, word[i:]...)
				break
			}
			newWord = append(newWord, word[i:j]...)
			i = j
			if word[i] == first && i < len(word)-1 && word[i+1] == second {
				newWord = append(newWord, first+second)
				i += 2
			} else {
				newWord = append(newWord, word[i])
				i += 1
			}
		}
		word = newWord
		if len(word) == 1 {
			break
		} else {
			rankedPairs = encoder.getRankedPairs(word)
		}
	}
	encoder.cache.Add(text, word)
	return word
}

func (encoder *GPTEncoder) SplitWords(text *string) *[]string {
	splitLines := strings.SplitAfter(*text, "\n")
	words := make([]string, 0, len(*text)/3)
	for lineIdx := 0; lineIdx < len(splitLines); lineIdx++ {
		line := splitLines[lineIdx]
		for ; lineIdx < len(splitLines)-1; {
			if splitLines[lineIdx+1] == "\n" {
				line = line + "\n"
				lineIdx += 1
			} else {
				break
			}
		}
		idxes := encoder.pattern.FindAllStringIndex(line, -1)
		for idx := range idxes {
			words = append(words, line[idxes[idx][0]:idxes[idx][1]])
		}
	}
	return &words
}

func (encoder *GPTEncoder) Encode(text *string) *Tokens {
	words := encoder.SplitWords(text)
	encoded := make(Tokens, 0)
	for idx := range *words {
		fragment := encoder.toUnicode(&(*words)[idx])
		token := encoder.toBPE(fragment)
		encoded = append(encoded, encoder.encodeTokens(&token)...)
	}
	return &encoded
}

func (encoder *GPTEncoder) Decode(encoded *Tokens) (text string) {
	// First convert our `Token` tokens into an 8-bit byte array.
	bs := make([]byte, 0, len(*encoded))
	for idx := range *encoded {
		if v, ok := encoder.decoder[(*encoded)[idx]]; ok {
			for bIdx := range v {
				bs = append(bs, v[bIdx])
			}
		}
	}
	// Convert our bytearray to string, interpreting as UTF-8 and then to
	// 32-bit runes.
	runes := []rune(string(bs))
	decoded := make([]byte, len(runes))
	// Convert our runes into 8-bit bytes using a 256-slot lookup table.
	for runeIdx := range runes {
		decoded[runeIdx] = encoder.runeToByte[runes[runeIdx]]
	}
	// Decode our final representation into an Unicode string.
	text = string(decoded)
	return text
}

var Encoder = NewEncoder()
var blankString = ""
var _ = Encoder.Encode(&blankString)