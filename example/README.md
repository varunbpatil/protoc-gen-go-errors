# Example Go project
This is an example Go project to demonstrate how you can use the `protoc-gen-go-errors` plugin
to generate Go error types from protobuf.

## Prerequisites
You need to have [buf](https://buf.build/docs/cli/installation/),
[protoc-gen-go](https://grpc.io/docs/languages/go/quickstart/)
and `protoc-gen-go-errors` installed in your `PATH`.

To install `protoc-gen-go-errors`, you can run:
```sh
go install github.com/varunbpatil/protoc-gen-go-errors@latest
```

You also need to manually download [options.proto](../proto/errors/options.proto)
and place it in the appropriate path. In this example Go project, it is placed
in `proto/errors/options.proto`. Don't forget to update the `go_package` to
match your Go project repo structure.

> [!IMPORTANT]  
> Changing the (errors.display) option tag (51234) in `options.proto` is currently not supported.
> If you change it, you'll see an error that looks like this:
> ```
> Missing (errors.Display) option in message ...
> ```

## Generating code
Run the following command:
```sh
buf generate
```
This will generate the Go code for errors in the `gen/` directory which you can then import in your code.

## (Optional) Development environment
This particular example project uses [devbox](https://www.jetify.com/devbox) to generate an isolated
development environment in which [buf](https://github.com/bufbuild/buf), 
[protoc-gen-go](https://pkg.go.dev/github.com/golang/protobuf/protoc-gen-go) and
protoc-gen-go-errors are installed.

It is definitely recommended to use [buf](https://github.com/bufbuild/buf) for working with
protobufs, but you don't necessarily need to use [devbox](https://www.jetify.com/devbox)
to be able to work with `protoc-gen-go-errors`. You just have to ensure that you have both
`protoc-gen-go` and `protoc-gen-go-errors` installed and available in your `PATH`. The protoc
compile will find the executables from the `PATH`.
