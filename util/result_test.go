package util_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/varunbpatil/protoc-gen-go-errors/test/example.com/test/errorspb"
	g "github.com/varunbpatil/protoc-gen-go-errors/util"
)

func TestResult(t *testing.T) {
	t.Parallel()

	// Config error.
	r := configError()
	require.True(t, r.IsErr())
	require.False(t, r.IsOK())
	require.Equal(t, "invalid config: host=localhost", r.Err().Error())

	// Get returns the zero value and the error on failure.
	v, err := r.Get()
	require.Empty(t, v)
	require.Equal(t, "invalid config: host=localhost", err.Error())

	// Fallbacks are used on failure.
	require.Equal(t, "fallback", r.OrElse("fallback"))
	require.Empty(t, r.OrEmpty())
	require.Panics(t, func() { _ = r.MustGet() })

	// No error.
	r = configNoError()
	require.True(t, r.IsOK())
	require.False(t, r.IsErr())
	require.Equal(t, "localhost", r.MustGet())
	require.Equal(t, "localhost", r.OrElse("fallback"))
	require.Equal(t, "localhost", r.OrEmpty())
	require.Nil(t, r.Err())

	// Get returns the stored value on success.
	v, err = r.Get()
	require.Equal(t, "localhost", v)
	require.Nil(t, err)
}

func configError() g.Result[string, *errorspb.ConfigError] {
	return g.Err[string](&errorspb.ConfigError{
		Key:     "host",
		Value:   "localhost",
		Message: "the provided host is invalid",
	})
}

func configNoError() g.Result[string, *errorspb.ConfigError] {
	return g.Ok[string, *errorspb.ConfigError]("localhost")
}
