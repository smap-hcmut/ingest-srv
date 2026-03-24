package usecase

import "testing"

func TestSupportsParse(t *testing.T) {
	uc := &implUseCase{}

	if !uc.SupportsParse("tiktok", "full_flow") {
		t.Fatalf("expected tiktok/full_flow to be supported")
	}
	if !uc.SupportsParse(" TiKtOk ", " FULL_FLOW ") {
		t.Fatalf("expected SupportsParse to be case-insensitive and trim spaces")
	}
	if uc.SupportsParse("facebook", "full_flow") {
		t.Fatalf("expected facebook/full_flow to be unsupported in phase 2")
	}
	if uc.SupportsParse("tiktok", "comments") {
		t.Fatalf("expected tiktok/comments to be unsupported in phase 2")
	}
}

func TestResolveParser(t *testing.T) {
	uc := &implUseCase{}

	parser, ok := uc.resolveParser("tiktok", "full_flow")
	if !ok {
		t.Fatalf("expected parser lookup to succeed for tiktok/full_flow")
	}
	if parser == nil {
		t.Fatalf("expected non-nil parser for tiktok/full_flow")
	}

	parser, ok = uc.resolveParser("youtube", "full_flow")
	if ok {
		t.Fatalf("expected parser lookup to fail for youtube/full_flow in phase 2")
	}
	if parser != nil {
		t.Fatalf("expected nil parser for unsupported target")
	}
}
