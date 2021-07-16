package gpt_bpe

import (
	"github.com/jdkato/prose/v2"
	"strings"
	"unicode"
)

type TrimDirection uint

const (
	TrimTop TrimDirection = iota
	TrimBottom TrimDirection = iota
	TrimNone TrimDirection = iota
)

func (encoder GPTEncoder) TrimSentences(tokens *[]uint16, direction TrimDirection,
	limit uint) (*[]uint16, error) {
	var err error
	trimmed := make([]uint16, 0)
	if uint(len(*tokens)) <= limit {
		return tokens, err
	} else if direction == TrimNone {
		return &trimmed, err
	}
	doc, err := prose.NewDocument(encoder.Decode(tokens))
	if err != nil {
		return &trimmed, err
	}
	sentences := doc.Sentences()
	var start, end, step, idx int
	var textBegin, textEnd int
	var sentenceIdx int
	switch direction {
	case TrimTop:
		start = len(sentences)-1
		end = -1
		step = -1
		textBegin = 0
		textEnd = len(doc.Text)
	case TrimBottom:
		start = 0
		end = len(sentences)
		step = 1
		textBegin = 0
		textEnd = len(doc.Text)
	default:
		return &trimmed, err
	}
	for idx = start; idx != end; idx += step {
		sentence := sentences[idx].Text
		switch direction {
		case TrimTop:
			sentenceIdx = strings.LastIndex(doc.Text[textBegin:textEnd], sentence)+textBegin
			if sentenceIdx > 0 && unicode.IsSpace(rune(doc.Text[sentenceIdx])) {
				sentenceIdx -= 1
			}
			toTokenize := doc.Text[sentenceIdx:]
			tokCt := uint(len(*(encoder.Encode(&toTokenize))))
			if tokCt >= limit {
				toEncode := doc.Text[textEnd:]
				return encoder.Encode(&toEncode), err
			}
			textEnd = sentenceIdx - 1
		case TrimBottom:
			sentenceIdx = strings.Index(doc.Text[textBegin:textEnd], sentence)+textBegin
			toTokenize := doc.Text[0:sentenceIdx+len(sentence)]
			tokCt := uint(len(*(encoder.Encode(&toTokenize))))
			if tokCt >= limit {
				toEncode := doc.Text[0:textEnd]
				return encoder.Encode(&toEncode), err
			}
			textBegin += len(sentence) + 1
		}
	}
	return &trimmed, err
}