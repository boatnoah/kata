package tui

import "testing"

func TestAIStream_AppendDeltaAccumulates(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("Hello")
	s.AppendDelta(", ")
	s.AppendDelta("world")
	if got, want := s.Buffer(), "Hello, world"; got != want {
		t.Fatalf("Buffer() = %q, want %q", got, want)
	}
	if s.Rendered() != "" {
		t.Fatalf("Rendered() = %q, want empty (no advance yet)", s.Rendered())
	}
}

func TestAIStream_ReplaceBufferOverwrites(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("partial draft ")
	s.ReplaceBuffer("final text from completion event")
	if got, want := s.Buffer(), "final text from completion event"; got != want {
		t.Fatalf("Buffer() after ReplaceBuffer = %q, want %q", got, want)
	}
}

func TestAIStream_SetRenderedWriteback(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("abcdef")
	s.SetRendered("abc")
	if got, want := s.Rendered(), "abc"; got != want {
		t.Fatalf("Rendered() = %q, want %q", got, want)
	}
	if s.Buffer() != "abcdef" {
		t.Fatalf("Buffer() should be unchanged by SetRendered, got %q", s.Buffer())
	}
}

func TestAIStream_SetLabelFlips(t *testing.T) {
	s := newAIStream(TranscriptThinking)
	if s.Label() != TranscriptThinking {
		t.Fatalf("initial Label() = %q, want %q", s.Label(), TranscriptThinking)
	}
	s.SetLabel(TranscriptAssistant)
	if s.Label() != TranscriptAssistant {
		t.Fatalf("Label() after SetLabel = %q, want %q", s.Label(), TranscriptAssistant)
	}
}
