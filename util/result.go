// Inspired by https://github.com/samber/mo/blob/master/result.go

package util

type Result[T any, E error] struct {
	isErr bool
	value T
	err   E
}

func Ok[T any, E error](value T) Result[T, E] {
	return Result[T, E]{
		value: value,
		isErr: false,
	}
}

func Err[T any, E error](err E) Result[T, E] {
	return Result[T, E]{
		err:   err,
		isErr: true,
	}
}

func (r Result[T, E]) IsOK() bool {
	return !r.isErr
}

func (r Result[T, E]) IsErr() bool {
	return r.isErr
}

func (r Result[T, E]) Get() (T, E) {
	return empty[T](), r.err
}

func (r Result[T, E]) MustGet() T {
	if r.isErr {
		panic(r.err)
	}
	return r.value
}

func (r Result[T, E]) OrElse(fallback T) T {
	if r.isErr {
		return fallback
	}
	return r.value
}

func (r Result[T, E]) OrEmpty() T {
	return r.value
}

func (r Result[T, E]) Err() E {
	return r.err
}

func empty[T any]() (t T) {
	return
}
