// Package embedtext normalizes free-form slot values before they are
// concatenated into the text we feed to the embeddings model.
//
// The lost-report / found-item embedding texts are produced by joining a
// fixed list of slots (item_name, brand, color, ...) with " | ". In practice
// many of those slots come back from the chat agent as either the literal
// word "unknown" or a boiler-plate phrase like "Copperish in color" or
// "Turn on on brand." Passing those through verbatim adds a lot of
// low-information tokens to the embedding and pushes true-positive matches
// below any reasonable cosine threshold.
//
// CleanSlotValue returns "" for values that should be dropped entirely, and
// strips trailing boiler-plate such as " in color" / " for brand" when the
// slot-name hint matches. JoinNonEmpty is a convenience that applies the
// cleaner across every slot and joins the survivors with " | ".
package embedtext

import (
	"regexp"
	"strings"
)

// Slot is a short tag that tells the cleaner which trailing filler phrases
// may legitimately be stripped from a value. The empty Slot skips trailing
// filler stripping and only drops unknown / boiler-plate sentinels.
type Slot string

const (
	SlotItemName        Slot = "item_name"
	SlotItemDescription Slot = "item_description"
	SlotItemType        Slot = "item_type"
	SlotBrand           Slot = "brand"
	SlotModel           Slot = "model"
	SlotColor           Slot = "color"
	SlotMaterial        Slot = "material"
	SlotItemCondition   Slot = "item_condition"
	SlotCategory        Slot = "category"
	SlotLocation        Slot = "location"
	SlotRoute           Slot = "route"
	SlotRouteID         Slot = "route_id"
)

// unknownSentinels is the set of exact (case-insensitive) values the chat
// agent or staff form may send when a slot is unknown. Anything matching
// one of these is dropped from the embedding text entirely.
var unknownSentinels = map[string]struct{}{
	"":        {},
	"unknown": {},
	"n/a":     {},
	"na":      {},
	"none":    {},
	"null":    {},
	"-":       {},
	"?":       {},
}

// slotFillerPatterns is a per-slot list of trailing boiler-plate phrases we
// strip off the end of the value. The patterns are anchored to the end of
// the string and are matched case-insensitively.
var slotFillerPatterns = map[Slot][]*regexp.Regexp{
	SlotColor: {
		regexp.MustCompile(`(?i)\s+(?:in|for|of)\s+colou?r\s*$`),
		regexp.MustCompile(`(?i)\s+colou?red\s*$`),
		regexp.MustCompile(`(?i)\s+colou?r\s*$`),
	},
	SlotBrand: {
		regexp.MustCompile(`(?i)\s+(?:is\s+the\s+)?brand\s*\.?\s*$`),
		regexp.MustCompile(`(?i)^(?:the\s+)?brand\s+is\s+`),
	},
	SlotModel: {
		regexp.MustCompile(`(?i)\s+(?:is\s+the\s+)?model\s*\.?\s*$`),
	},
	SlotMaterial: {
		regexp.MustCompile(`(?i)\s+material\s*\.?\s*$`),
		regexp.MustCompile(`(?i)\s+made\s+of\s+`),
	},
	SlotItemType: {
		regexp.MustCompile(`(?i)\s+(?:item\s+)?type\s*\.?\s*$`),
	},
	SlotCategory: {
		regexp.MustCompile(`(?i)\s+category\s*\.?\s*$`),
	},
}

// multiSpace collapses runs of whitespace into a single space.
var multiSpace = regexp.MustCompile(`\s+`)

// CleanSlotValue normalizes a single slot value. It returns "" when the
// value is effectively empty / unknown and therefore should not be added
// to the embedding text.
func CleanSlotValue(slot Slot, value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}

	// Strip a trailing period so "unknown." still matches the sentinel set.
	trimmed := strings.TrimRight(v, " .")
	if _, ok := unknownSentinels[strings.ToLower(trimmed)]; ok {
		return ""
	}

	if patterns, ok := slotFillerPatterns[slot]; ok {
		for _, p := range patterns {
			v = p.ReplaceAllString(v, "")
		}
	}

	v = multiSpace.ReplaceAllString(v, " ")
	v = strings.TrimSpace(v)
	v = strings.TrimRight(v, " .,;:")
	v = strings.TrimSpace(v)

	// Re-check the sentinel set after filler stripping (e.g. a value of
	// just "brand" after removing "is the ... brand").
	if v == "" {
		return ""
	}
	if _, ok := unknownSentinels[strings.ToLower(v)]; ok {
		return ""
	}

	return v
}

// JoinNonEmpty applies CleanSlotValue to every (slot, value) pair in order
// and joins the survivors with " | ". Pairs whose cleaned value is empty
// are skipped, which is how we keep literal "unknown" tokens from
// polluting the embedding.
func JoinNonEmpty(pairs []Pair) string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if v := CleanSlotValue(p.Slot, p.Value); v != "" {
			out = append(out, v)
		}
	}
	return strings.Join(out, " | ")
}

// Pair is a single (slot-tag, raw-value) entry passed to JoinNonEmpty.
type Pair struct {
	Slot  Slot
	Value string
}
