module example.com/test

go 1.27.1

require (
	github.com/stretchr/testify v1.12.1
	github.com/varunbpatil/protoc-gen-go-errors v0.0.0
	google.golang.org/protobuf v1.36.12
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

// The example consumes the plugin's error options (errors/options.proto) from
// this repository, mirroring how a real downstream project would depend on it.
replace github.com/varunbpatil/protoc-gen-go-errors => ../
