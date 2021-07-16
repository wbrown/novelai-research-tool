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

type GPTEncoder struct {
	encoder    map[string]uint16
	decoder    map[uint16][]byte
	bpe_ranks  map[GPTPair]float64
	pattern    *regexp.Regexp
	byteToRune map[byte]rune
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
	encoderTokens := make(map[string]uint16)
	json.Unmarshal(encoderFile, &encoderTokens)
	tokensEncoder := make(map[uint16][]byte)
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
	bs := make([]uint8, 0)
	bytesUnicode := make(map[byte]rune)
	unicodeBytes := make(map[rune]byte)
	for b := uint8('!'); b < uint8('~')+1; b++ {
		bs = append(bs, b)
		bytesUnicode[b] = rune(b)
		unicodeBytes[rune(b)] = b
	}
	for b := uint8('¡'); b < uint8('¬')+1; b++ {
		bs = append(bs, b)
		bytesUnicode[b] = rune(b)
		unicodeBytes[rune(b)] = b
	}
	for b := uint16('®'); b < uint16('ÿ')+1; b++ {
		bs = append(bs, byte(b))
		bytesUnicode[byte(b)] = rune(b)
		unicodeBytes[rune(b)] = byte(b)
	}
	uct := 0
	for b := uint16(0); b < 256; b++ {
		if _, ok := bytesUnicode[uint8(b)]; !ok {
			bytesUnicode[uint8(b)] = rune(256 + uct)
			unicodeBytes[rune(256+uct)] = uint8(b)
			uct += 1
		}
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

func getPairs(word []string) []GPTPair {
	pairsSet := make(map[GPTPair]bool, 0)
	pairs := make([]GPTPair, 0)
	begin := 1
	prev := word[0]
	for idx := begin; idx < len(word); idx++ {
		present := word[idx]
		pair := GPTPair{prev, present}
		if _, ok := pairsSet[pair]; !ok {
			pairs = append(pairs, pair)
		}
		pairsSet[pair] = true
		prev = present
	}
	return pairs
}

func (encoder *GPTEncoder) rankPairs(pairs []GPTPair) BGERanks {
	rankedPairs := make(BGERanks, 0)
	for idx := range pairs {
		bpe, ok := encoder.bpe_ranks[pairs[idx]]
		if !ok {
			bpe = math.Inf(1)
		}
		rankedPairs = append(rankedPairs, BGERank{bpe, pairs[idx]})
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
	outArr := make([]rune, 0)
	textBytes := []byte(*text)
	for idx := range textBytes {
		outArr = append(outArr, encoder.byteToRune[textBytes[idx]])
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

func (encoder *GPTEncoder) encodeTokens(tokens *[]string) (encoded []uint16) {
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
	pairs := getPairs(word)
	if len(pairs) == 0 {
		return []string{text}
	}
	for {
		bigram := encoder.minPair(pairs)
		if _, ok := encoder.bpe_ranks[bigram]; !ok {
			break
		}
		first := bigram.left
		second := bigram.right
		newWord := make([]string, 0)
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
			pairs = getPairs(word)
		}
	}
	encoder.cache.Add(text, word)
	return word
}

func (encoder *GPTEncoder) SplitWords(text *string) *[]string {
	idxes := encoder.pattern.FindAllStringIndex(*text, -1)
	words := make([]string, 0)
	for idx := range idxes {
		words = append(words, (*text)[idxes[idx][0]:idxes[idx][1]])
	}
	return &words
}

func (encoder *GPTEncoder) Encode(text *string) *[]uint16 {
	words := encoder.SplitWords(text)
	encoded := make([]uint16, 0)
	for idx := range *words {
		fragment := encoder.toUnicode(&(*words)[idx])
		token := encoder.toBPE(fragment)
		encoded = append(encoded, encoder.encodeTokens(&token)...)
	}
	return &encoded
}

func (encoder *GPTEncoder) Decode(encoded *[]uint16) (text string) {
	// First convert our `uint16` tokens into an 8-bit byte array.
	bs := make([]byte, 0)
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
	decoded := make([]byte, 0)
	// Convert our runes into 8-bit bytes using a 256-slot lookup table.
	for runeIdx := range runes {
		decoded = append(decoded, encoder.runeToByte[runes[runeIdx]])
	}
	// Decode our final representation into an Unicode string.
	text = string(decoded)
	return text
}
