package gpt_bpe

import (
	"github.com/jdkato/prose/v2"
	"strings"
	"unicode"
)

type TrimDirection uint

const (
	TrimTop    TrimDirection = iota
	TrimBottom TrimDirection = iota
	TrimNone   TrimDirection = iota
)

func (encoder GPTEncoder) TrimNewlines(tokens *[]uint16, direction TrimDirection,
	limit uint) (*[]uint16, error) {
	var err error
	trimmed := make([]uint16, 0)
	if uint(len(*tokens)) <= limit {
		return tokens, err
	} else if direction == TrimNone {
		return &trimmed, err
	}
	lines := strings.Split(encoder.Decode(tokens), "\n")
	var start, end, step, idx int
	switch direction {
	case TrimTop:
		start = len(lines) - 1
		end = -1
		step = -1
	case TrimBottom:
		start = 0
		end = len(lines)
		step = 1
	}
	accTokens := make([]uint16, 0)
	for idx = start; idx != end; idx += step {
		line := lines[idx]
		switch direction {
		case TrimTop:
			line = "\n" + line
		case TrimBottom:
			line = line + "\n"
		}
		newTokens := encoder.Encode(&line)
		if len(*newTokens)+len(accTokens) > int(limit) {
			return &accTokens, err
		} else {
			switch direction {
			case TrimTop:
				accTokens = append(*newTokens, accTokens...)
			case TrimBottom:
				accTokens = append(accTokens, *newTokens...)
			}
		}
	}
	return &accTokens, err
}

func (encoder GPTEncoder) TrimSentences(tokens *[]uint16, direction TrimDirection,
	limit uint) (*[]uint16, error) {
	var err error
	trimmed := make([]uint16, 0)
	if uint(len(*tokens)) <= limit {
		return tokens, err
	} else if direction == TrimNone {
		return &trimmed, err
	}
	doc, err := prose.NewDocument(encoder.Decode(tokens),
		prose.WithTagging(false),
		prose.WithExtraction(false),
		prose.WithTokenization(false))
	if err != nil {
		return &trimmed, err
	}
	sentences := doc.Sentences()
	var start, end, step, idx int
	var textBegin, textEnd int
	var sentenceIdx, lastSentence int
	switch direction {
	case TrimTop:
		start = len(sentences) - 1
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
			sentenceIdx = strings.LastIndex(doc.Text[textBegin:],
				sentence) + textBegin
			if sentenceIdx > 0 && sentenceIdx < len(doc.Text) &&
				unicode.IsSpace(rune(doc.Text[sentenceIdx])) {
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
			sentenceIdx = strings.Index(doc.Text[textBegin:textEnd],
				sentence) + textBegin
			sentenceEnd := sentenceIdx + len(sentence)
			if sentenceEnd < textEnd &&
				doc.Text[sentenceEnd:sentenceEnd+1] == "\n" {
				sentenceEnd += 1
			}
			toTokenize := doc.Text[0:sentenceEnd]
			tokCt := uint(len(*(encoder.Encode(&toTokenize))))
			if tokCt >= limit {
				toEncode := doc.Text[0:lastSentence]
				return encoder.Encode(&toEncode), err
			}
			lastSentence = sentenceEnd
			textBegin += len(sentence)
		}
	}
	return &trimmed, err
}
