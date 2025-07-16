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
	require.Equal(t, "invalid config: host=localhost", r.Err().Error())

	// No error.
	r = configNoError()
	require.True(t, r.IsOK())
	require.Equal(t, "localhost", r.MustGet())
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
