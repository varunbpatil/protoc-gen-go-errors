package errors_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/varunbpatil/protoc-gen-go-errors/test/example.com/test/errorspb"
)

func TestApplicationErrorConfigError(t *testing.T) {
	t.Parallel()

	// Create a config error.
	cErr := &errorspb.ConfigError{
		Key:     "host",
		Value:   "localhost",
		Message: "the provided host is invalid",
	}
	require.Equal(t, "invalid config: host=localhost", cErr.Error())

	// Wrap the config error in an application error.
	aErr := new(errorspb.ApplicationError).FromConfigError(cErr)
	require.Equal(t, "invalid config: host=localhost", aErr.Error())

	// Unwrap the config error from the application error.
	var cErrUnwrapped *errorspb.ConfigError
	require.ErrorAs(t, aErr, &cErrUnwrapped)
	require.Equal(t, cErr, cErrUnwrapped)

	// No further errors to unwrap.
	require.Nil(t, cErrUnwrapped.Unwrap())
}

func TestApplicationErrorIOError(t *testing.T) {
	t.Parallel()

	// Create an I/O error.
	iErr := &errorspb.IOError{
		Path: "file.txt",
		Cause: &errorspb.NotFoundError{
			Entity: "file",
		},
	}
	require.Equal(t, "could not read file.txt: not found: file", iErr.Error())

	// Wrap the I/O error in an application error.
	aErr := new(errorspb.ApplicationError).FromIOError(iErr)
	require.Equal(t, "could not read file.txt: not found: file", aErr.Error())

	// Unwrap the I/O error from the application error.
	var iErrUnwrapped *errorspb.IOError
	require.ErrorAs(t, aErr, &iErrUnwrapped)
	require.Equal(t, iErr, iErrUnwrapped)

	// Unwrap the not found error from the I/O error.
	var nErrUnwrapped *errorspb.NotFoundError
	require.ErrorAs(t, iErr, &nErrUnwrapped)
	require.Equal(t, &iErr.Cause, &nErrUnwrapped)

	// No further errors to unwrap.
	require.Nil(t, nErrUnwrapped.Unwrap())
}

func TestApplicationErrorOtherError(t *testing.T) {
	t.Parallel()

	// Create an "other" error.
	oErr := &errorspb.OtherError{
		Message: "something went wrong",
	}
	require.Equal(t, "something went wrong", oErr.Error())

	// Wrap the "other" error in an application error.
	aErr := new(errorspb.ApplicationError).FromOtherError(oErr)
	require.Equal(t, "something went wrong", aErr.Error())

	// Unwrap the "other" error from the application error.
	var oErrUnwrapped *errorspb.OtherError
	require.ErrorAs(t, aErr, &oErrUnwrapped)
	require.Equal(t, oErr, oErrUnwrapped)

	// No further errors to unwrap.
	require.Nil(t, oErrUnwrapped.Unwrap())
}
