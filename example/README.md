# Example Go project
This is an example Go project to demonstrate how you can use the `protoc-gen-go-errors` plugin
to generate Go error types from protobuf.

## Prerequisites
You need to have [buf](https://buf.build/docs/cli/installation/),
[protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go)
and `protoc-gen-go-errors` installed in your `PATH`.

To install `protoc-gen-go-errors`, you can run:
```sh
go install github.com/varunbpatil/protoc-gen-go-errors@latest
```

Inside this repository, `options.proto` is defined exactly once at
[`protos/protoc-gen-go-errors/options.proto`](../protos/protoc-gen-go-errors/options.proto)
and imported here through the buf workspace in the root [`buf.yaml`](../buf.yaml)
— no copy is needed. If you're using the plugin in your own project, see
[the root README](../README.md) for how to obtain `options.proto` without
manually copying it.

> [!IMPORTANT]  
> Changing the (errors.display) option tag (51234) in `options.proto` is currently not supported.
> If you change it, you'll see an error that looks like this:
> ```
> Missing (errors.display) option in message ...
> ```

## Generating code
From the repository root, run:
```sh
mise run generate
```
This builds `protoc-gen-go-errors` from the current source tree into
`example/bin`, puts it on the `PATH` via mise, and runs `buf generate` (which
regenerates all three modules: `errors/`, `test/` and `example/`). This
example's generated code lands in the `gen/` directory, which you can then
import in your code.

## (Optional) Development environment
The repository uses [mise](https://mise.jdx.dev) to create an isolated
development environment in which [Go](https://go.dev), [buf](https://github.com/bufbuild/buf),
[protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go),
[golangci-lint](https://golangci-lint.run) and a locally-built
`protoc-gen-go-errors` are installed. The tool versions are pinned in the root
[`mise.toml`](../mise.toml), so this example shares the same environment.

Set up the environment with:
```sh
mise trust
mise install
```
Then run the tasks (`mise run ...`) as described above. `mise run test` runs
this example's tests. You can execute the tasks from anywhere inside the
repository — mise resolves them from the root configuration.

You don't need to use mise to work with `protoc-gen-go-errors`. You just have
to ensure that `protoc-gen-go` and `protoc-gen-go-errors` are installed and
available in your `PATH` before running `buf generate`.
