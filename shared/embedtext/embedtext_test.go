package embedtext

import "testing"

func TestCleanSlotValue_DropsUnknownSentinels(t *testing.T) {
	cases := []struct{ slot Slot; in string }{
		{SlotBrand, "unknown"},
		{SlotBrand, "Unknown"},
		{SlotColor, "UNKNOWN"},
		{SlotModel, "unknown."},
		{SlotCategory, "n/a"},
		{SlotMaterial, "none"},
		{SlotItemType, ""},
		{SlotColor, "   "},
		{SlotModel, "-"},
	}
	for _, c := range cases {
		if got := CleanSlotValue(c.slot, c.in); got != "" {
			t.Errorf("CleanSlotValue(%q, %q) = %q, want \"\"", c.slot, c.in, got)
		}
	}
}

func TestCleanSlotValue_StripsTrailingFiller(t *testing.T) {
	cases := []struct {
		slot     Slot
		in, want string
	}{
		{SlotColor, "Copperish in color", "Copperish"},
		{SlotColor, "copperish coloured", "copperish"},
		{SlotBrand, "Pixel is the brand.", "Pixel"},
		{SlotBrand, "Turn on on brand.", "Turn on on"},
		{SlotModel, "7a is the model.", "7a"},
		{SlotMaterial, "leather material.", "leather"},
		{SlotItemType, "headphones item type.", "headphones"},
		{SlotCategory, "Electronics category", "Electronics"},
		{SlotItemName, "Keys", "Keys"},
		{SlotItemDescription, "There's a lanyard attached.", "There's a lanyard attached"},
	}
	for _, c := range cases {
		if got := CleanSlotValue(c.slot, c.in); got != c.want {
			t.Errorf("CleanSlotValue(%q, %q) = %q, want %q", c.slot, c.in, got, c.want)
		}
	}
}

func TestJoinNonEmpty_DropsFillerTokens(t *testing.T) {
	// Mirrors the actual lost_report row that triggered the bug report.
	got := JoinNonEmpty([]Pair{
		{Slot: SlotItemName, Value: "Keys"},
		{Slot: SlotItemDescription, Value: "There's a lanyard attached to the keys."},
		{Slot: SlotItemType, Value: "unknown"},
		{Slot: SlotBrand, Value: "Turn on on brand."},
		{Slot: SlotModel, Value: "unknown"},
		{Slot: SlotColor, Value: "Copperish in color"},
		{Slot: SlotMaterial, Value: "unknown"},
		{Slot: SlotItemCondition, Value: "unknown"},
		{Slot: SlotCategory, Value: "unknown"},
		{Slot: SlotLocation, Value: "unknown"},
		{Slot: SlotRoute, Value: "Monroe -> Ruston"},
		{Slot: SlotRouteID, Value: ""},
	})
	want := "Keys | There's a lanyard attached to the keys | Turn on on | Copperish | Monroe -> Ruston"
	if got != want {
		t.Errorf("JoinNonEmpty =\n  %q\nwant\n  %q", got, want)
	}
}
