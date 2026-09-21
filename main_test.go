package main

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlaceholderNames(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{"key", "value"}, placeholderNames("invalid: {key}={value}"))
	require.Equal(t, []string{"a", "a"}, placeholderNames("{a} and {a}"))
	require.Empty(t, placeholderNames("no placeholders here"))
}

func TestToPrintfFormat(t *testing.T) {
	t.Parallel()

	require.Equal(t, "invalid: %v=%v", toPrintfFormat("invalid: {key}={value}"))

	// Literal percent signs must be escaped so fmt does not interpret them.
	require.Equal(t, "50%% off %v", toPrintfFormat("50% off {item}"))
	require.Equal(t, "100%% done", toPrintfFormat("100% done"))

	// Braces that are not valid placeholders are left untouched.
	require.Equal(t, "{} not a field", toPrintfFormat("{} not a field"))
}

func TestQuotedDisplayFormat(t *testing.T) {
	t.Parallel()

	// The format is embedded in generated source as a Go string literal.
	require.Equal(t, `"he said \"%v\""`, strconv.Quote(toPrintfFormat(`he said "{quote}"`)))
}
