package structs

import gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"

type BiasType uint

const (
	BiasString    BiasType = 0
	BiasTokens             = 1
	BiasLitString          = 2
)

type BiasSequences struct {
	Sequences []gpt_bpe.Tokens `json:"sequences"`
	Type      BiasType         `json:"type"`
}

type BiasGroup struct {
	YamlPhrases          *[]string        `json:"-" yaml:"phrases"`
	Phrases              *[]BiasSequences `json:"phrases,omitempty" yaml:"-"`
	Bias                 *float64         `json:"bias,omitempty" yaml:"bias"`
	EnsureSequenceFinish *bool            `json:"ensure_sequence_finish,omitempty" yaml:"ensureSequenceFinish"`
	GenerateOnce         *bool            `json:"generate_once,omitempty" yaml:"generateOnce"`
	Enabled              *bool            `json:"enabled,omitempty" yaml:"enabled"`
	WhenInactive         *bool            `json:"whenInactive,omitempty" yaml:"whenInactive"`
}

type BiasGroups []BiasGroup

func (biasGroups *BiasGroups) RealizeBiases() {
	for biasIdx := range *biasGroups {
		biasGroup := (*biasGroups)[biasIdx]
		if biasGroup.YamlPhrases != nil {
			if (*biasGroups)[biasIdx].Phrases == nil {
				biasSequences := make([]BiasSequences, 0)
				(*biasGroups)[biasIdx].Phrases = &biasSequences
			}
			for phraseIdx := range *biasGroup.YamlPhrases {
				jsonifiedPhrase := BiasSequences{
					Sequences: make([]gpt_bpe.Tokens, 0),
					Type:      BiasLitString,
				}
				phraseString := (*biasGroup.YamlPhrases)[phraseIdx]
				tokens := gpt_bpe.Encoder.Encode(&phraseString)
				jsonifiedPhrase.Sequences = append(jsonifiedPhrase.Sequences,
					*tokens)
				*(*biasGroups)[biasIdx].Phrases = append(
					*(*biasGroups)[biasIdx].Phrases, jsonifiedPhrase)
			}
		}
	}
}
