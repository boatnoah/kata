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

func TestAIStream_CompleteWithFinalTextReplacesBuffer(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("partial draft ")
	s.Complete("final text from completion event")
	if got, want := s.Buffer(), "final text from completion event"; got != want {
		t.Fatalf("Buffer() after Complete(final) = %q, want %q", got, want)
	}
	if !s.IsCompleted() {
		t.Fatalf("Complete() should set IsCompleted() true")
	}
}

func TestAIStream_CompleteEmptyTextPreservesBuffer(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("accumulated streaming text")
	s.Complete("")
	if got, want := s.Buffer(), "accumulated streaming text"; got != want {
		t.Fatalf("Complete(\"\") should preserve buffer; got %q want %q", got, want)
	}
	if !s.IsCompleted() {
		t.Fatalf("Complete(\"\") should still set IsCompleted() true")
	}
}

func TestAIStream_CompletedToggle(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	if s.IsCompleted() {
		t.Fatalf("new stream should not be completed")
	}
	s.SetCompleted(true)
	if !s.IsCompleted() {
		t.Fatalf("SetCompleted(true) should flip flag on")
	}
	s.SetCompleted(false)
	if s.IsCompleted() {
		t.Fatalf("SetCompleted(false) should flip flag off")
	}
}

func TestAIStream_ResetClearsBufferAndRendered(t *testing.T) {
	s := newAIStream(TranscriptTool)
	s.AppendDelta("prior streamed text")
	s.Advance(99)
	s.Reset()
	if s.Buffer() != "" {
		t.Fatalf("Reset() should clear buffer, got %q", s.Buffer())
	}
	if s.Rendered() != "" {
		t.Fatalf("Reset() should clear rendered, got %q", s.Rendered())
	}
}

func TestAIStream_UpdateToolEmptyPreservesPrior(t *testing.T) {
	s := newAIStream(TranscriptTool)
	s.UpdateTool(toolSummary{Title: "Read", Detail: "go.mod", State: ToolPending})
	// Subsequent update with empty title/detail should preserve prior.
	s.UpdateTool(toolSummary{State: ToolOK})
	title, detail, state := s.Tool()
	if title != "Read" {
		t.Fatalf("title should be preserved on empty UpdateTool; got %q", title)
	}
	if detail != "go.mod" {
		t.Fatalf("detail should be preserved on empty UpdateTool; got %q", detail)
	}
	if state != ToolOK {
		t.Fatalf("state should update; got %v want %v", state, ToolOK)
	}
}

func TestAIStream_UpdateToolNonEmptyOverwrites(t *testing.T) {
	s := newAIStream(TranscriptTool)
	s.UpdateTool(toolSummary{Title: "Read", Detail: "go.mod", State: ToolPending})
	s.UpdateTool(toolSummary{Title: "Read again", Detail: "main.go", State: ToolOK})
	title, detail, state := s.Tool()
	if title != "Read again" || detail != "main.go" || state != ToolOK {
		t.Fatalf("UpdateTool with non-empty values should overwrite; got (%q, %q, %v)", title, detail, state)
	}
}

func TestAIStream_AdvanceEmptyBufferNoOp(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.Advance(3)
	if s.Rendered() != "" {
		t.Fatalf("Advance on empty buffer should leave rendered empty, got %q", s.Rendered())
	}
}

func TestAIStream_AdvanceRevealsRunesPerTick(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("abcdefghij")
	s.Advance(3)
	if got, want := s.Rendered(), "abc"; got != want {
		t.Fatalf("after Advance(3), Rendered() = %q, want %q", got, want)
	}
	s.Advance(4)
	if got, want := s.Rendered(), "abcdefg"; got != want {
		t.Fatalf("after Advance(4), Rendered() = %q, want %q", got, want)
	}
}

func TestAIStream_AdvanceLocksInAtCompletion(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("abcde")
	// Request more than the buffer holds.
	s.Advance(99)
	if got, want := s.Rendered(), "abcde"; got != want {
		t.Fatalf("Advance past buffer should lock in full text, got %q want %q", got, want)
	}
}

func TestAIStream_AdvanceIdempotentPastCompletion(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("xyz")
	s.Advance(3)
	s.Advance(3)
	s.Advance(99)
	if got, want := s.Rendered(), "xyz"; got != want {
		t.Fatalf("Advance after completion should be idempotent, got %q want %q", got, want)
	}
}

func TestAIStream_AdvanceIsRuneSafe(t *testing.T) {
	// Multi-byte runes should advance by codepoint, not byte.
	s := newAIStream(TranscriptAssistant)
	s.AppendDelta("héllo")
	s.Advance(2)
	if got, want := s.Rendered(), "hé"; got != want {
		t.Fatalf("Advance(2) on multi-byte buffer = %q, want %q", got, want)
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

func TestAIStream_TickingToggle(t *testing.T) {
	s := newAIStream(TranscriptAssistant)
	if s.IsTicking() {
		t.Fatalf("new stream should not be ticking")
	}
	s.SetTicking(true)
	if !s.IsTicking() {
		t.Fatalf("expected ticking after SetTicking(true)")
	}
	s.SetTicking(false)
	if s.IsTicking() {
		t.Fatalf("expected !ticking after SetTicking(false)")
	}
}

func TestAIStream_EnsureWaitingVerbIdempotent(t *testing.T) {
	s := newAIStream(TranscriptThinking)
	s.EnsureWaitingVerb()
	first := s.CurrentVerb()
	for i := 0; i < 10; i++ {
		s.EnsureWaitingVerb()
		if got := s.CurrentVerb(); got != first {
			t.Fatalf("EnsureWaitingVerb should be idempotent, iter %d got %q want %q", i, got, first)
		}
	}
}

func TestAIStream_AdvanceWaitingFrameCyclesDots(t *testing.T) {
	s := newAIStream(TranscriptThinking)
	dotsAt := func() string { return s.CurrentDots() }
	first := dotsAt()
	// Dots cycle every 6 frames; advancing 6 must change the dot string.
	for i := 0; i < 6; i++ {
		s.AdvanceWaitingFrame()
	}
	if dotsAt() == first {
		t.Fatalf("expected dots to cycle after 6 advances, still %q", first)
	}
}

func TestAIStream_ResetWaitingClearsVerbAndFrames(t *testing.T) {
	s := newAIStream(TranscriptThinking)
	s.EnsureWaitingVerb()
	for i := 0; i < 5; i++ {
		s.AdvanceWaitingFrame()
	}
	s.ResetWaiting()
	if s.CurrentDots() != spinnerDots[0] {
		t.Fatalf("ResetWaiting should reset frames; got dots %q want %q", s.CurrentDots(), spinnerDots[0])
	}
	// After reset, EnsureWaitingVerb should be allowed to pick again.
	s.EnsureWaitingVerb()
	// We can't assert a specific verb (random pick), only that calling
	// EnsureWaitingVerb again after reset doesn't panic and yields a
	// well-formed status.
	if s.WaitingStatus() == "" {
		t.Fatalf("WaitingStatus() should not be empty after reseed")
	}
}
