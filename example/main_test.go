package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"example.com/test/gen/app"
)

func TestApplicationErrorConfigError(t *testing.T) {
	t.Parallel()

	// Create a config error.
	cErr := &app.ConfigError{
		Key:     "host",
		Value:   "localhost",
		Message: "the provided host is invalid",
	}
	require.Equal(t, "invalid config: host=localhost", cErr.Error())

	// Wrap the config error in an application error.
	aErr := new(app.ApplicationError).FromConfigError(cErr)
	require.Equal(t, "invalid config: host=localhost", aErr.Error())

	// Unwrap the config error from the application error.
	var cErrUnwrapped *app.ConfigError
	require.ErrorAs(t, aErr, &cErrUnwrapped)
	require.Equal(t, cErr, cErrUnwrapped)

	// No further errors to unwrap.
	require.Nil(t, cErrUnwrapped.Unwrap())
}

func TestApplicationErrorIOError(t *testing.T) {
	t.Parallel()

	// Create an I/O error.
	iErr := &app.IOError{
		Path: "file.txt",
		Cause: &app.NotFoundError{
			Entity: "file",
		},
	}
	require.Equal(t, "could not read file.txt: not found: file", iErr.Error())

	// Wrap the I/O error in an application error.
	aErr := new(app.ApplicationError).FromIOError(iErr)
	require.Equal(t, "could not read file.txt: not found: file", aErr.Error())

	// Unwrap the I/O error from the application error.
	var iErrUnwrapped *app.IOError
	require.ErrorAs(t, aErr, &iErrUnwrapped)
	require.Equal(t, iErr, iErrUnwrapped)

	// Unwrap the not found error from the I/O error.
	var nErrUnwrapped *app.NotFoundError
	require.ErrorAs(t, iErr, &nErrUnwrapped)
	require.Equal(t, iErr.Cause, nErrUnwrapped)

	// No further errors to unwrap.
	require.Nil(t, nErrUnwrapped.Unwrap())
}

func TestApplicationErrorOtherError(t *testing.T) {
	t.Parallel()

	// Create an "other" error.
	oErr := &app.OtherError{
		Message: "something went wrong",
	}
	require.Equal(t, "something went wrong", oErr.Error())

	// Wrap the "other" error in an application error.
	aErr := new(app.ApplicationError).FromOtherError(oErr)
	require.Equal(t, "something went wrong", aErr.Error())

	// Unwrap the "other" error from the application error.
	var oErrUnwrapped *app.OtherError
	require.ErrorAs(t, aErr, &oErrUnwrapped)
	require.Equal(t, oErr, oErrUnwrapped)

	// No further errors to unwrap.
	require.Nil(t, oErrUnwrapped.Unwrap())
}

func TestErrorDisplayEscaping(t *testing.T) {
	t.Parallel()

	// A literal '%' in the display format must not be treated as a verb.
	pErr := &app.PercentError{Item: "apples"}
	require.Equal(t, "50% off apples", pErr.Error())

	// Quotes in the display format must survive code generation.
	qErr := &app.QuotedError{Quote: "hi"}
	require.Equal(t, `he said "hi"`, qErr.Error())
}

func TestUnwrapNilCause(t *testing.T) {
	t.Parallel()

	// An I/O error without a cause must not unwrap to a typed-nil error.
	iErr := &app.IOError{Path: "file.txt"}
	require.Nil(t, iErr.Unwrap())

	var nErr *app.NotFoundError
	require.False(t, errors.As(iErr, &nErr))

	// An application error without a kind must not unwrap either.
	aErr := &app.ApplicationError{}
	require.Nil(t, aErr.Unwrap())

	// Nor must one that wraps a nil leaf.
	aErr = &app.ApplicationError{Kind: &app.ApplicationError_Config{}}
	require.Nil(t, aErr.Unwrap())
}
