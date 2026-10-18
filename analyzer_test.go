package main

import (
	"errors"
	"testing"
)

func TestTextAnalyzerProcess(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AnalysisResult
		wantErr bool
	}{
		{
			name:  "english text",
			input: "Hello Go world",
			want:  AnalysisResult{WordCount: 3, CharacterCount: 14},
		},
		{
			name:  "russian text",
			input: "Привет мир",
			want:  AnalysisResult{WordCount: 2, CharacterCount: 10},
		},
		{
			name:  "multiple spaces",
			input: "Go   Go",
			want:  AnalysisResult{WordCount: 2, CharacterCount: 7},
		},
		{
			name:  "newline",
			input: "Go\nGo",
			want:  AnalysisResult{WordCount: 2, CharacterCount: 5},
		},
		{
			name:    "empty text",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   " \t\n ",
			wantErr: true,
		},
	}

	analyzer := TextAnalyzer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := analyzer.Process(tt.input)

			if tt.wantErr {
				if !errors.Is(err, ErrEmptyText) {
					t.Fatalf("expected ErrEmptyText, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}
