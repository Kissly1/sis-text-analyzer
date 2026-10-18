package main

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrEmptyText = errors.New("text must not be empty")

func validateText(text string) error {
	if strings.TrimSpace(text) == "" {
		return ErrEmptyText
	}

	return nil
}

type TextAnalyzer struct{}

func (TextAnalyzer) Process(text string) (AnalysisResult, error) {
	if err := validateText(text); err != nil {
		return AnalysisResult{}, err
	}

	result := AnalysisResult{
		WordCount:      len(strings.Fields(text)),
		CharacterCount: utf8.RuneCountInString(text),
	}

	return result, nil
}
