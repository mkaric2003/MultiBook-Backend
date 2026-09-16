package domain

import "testing"

func TestBusinessTextNormalization(t *testing.T) {
	t.Parallel()
	if got := TrimText("  Hotel Europe  "); got != "Hotel Europe" {
		t.Fatalf("TrimText() = %q", got)
	}
	if got := NormalizeSearchText("  SARAJEVO "); got != "sarajevo" {
		t.Fatalf("NormalizeSearchText() = %q", got)
	}
}

func TestOptionalBusinessTextNormalizationDoesNotMutateInput(t *testing.T) {
	t.Parallel()
	value := "  SARAJEVO  "
	normalized := NormalizeOptionalSearchText(&value)
	if normalized == nil || *normalized != "sarajevo" {
		t.Fatalf("NormalizeOptionalSearchText() = %#v", normalized)
	}
	if value != "  SARAJEVO  " {
		t.Fatalf("normalization mutated caller value: %q", value)
	}
}
