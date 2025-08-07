{
  description = "protoc-gen-go-errors";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixos-unstable";
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, gitignore }:
    let
      allSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        inherit system;
        pkgs =
          let
            pkgs-unstable = import nixpkgs-unstable { inherit system; };
          in
          import nixpkgs {
            inherit system;
            overlays = [
              (final: prev: {
                gopls = pkgs-unstable.gopls;
              })
            ];
          };
      });
    in
    {
      packages = forAllSystems ({ pkgs, ... }:
        rec {
          default = protoc-gen-go-errors;

          protoc-gen-go-errors = pkgs.buildGoModule {
            pname = "protoc-gen-go-errors";
            version = "0.1.0";

            src = gitignore.lib.gitignoreSource ./.;

            vendorHash = "sha256-+y5WBX89XcRPHHfX3Eiie97QydC3t2mjpkf6lDczBWo=";

            subPackages = [ "." ];

            ldflags = [
              "-s"
              "-w"
              "-extldflags=-static"
            ];

            env.CGO_ENABLED = "0";

            meta = with pkgs.lib; {
              description = "Protocol Buffer compiler plugin for generating Go error types";
              homepage = "https://github.com/varunbpatil/protoc-gen-go-errors";
              license = licenses.mit;
              maintainers = [ maintainers.varunbpatil ];
              platforms = platforms.unix;
            };
          };
        });

      devShells = forAllSystems ({ pkgs, ... }: {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            golangci-lint
            gopls
            protobuf
            protoc-gen-go
            buf
          ];

          shellHook = ''
            echo "Development environment for protoc-gen-go-errors"
            echo "Go version: $(go version)"
            echo "Protobuf version: $(protoc --version)"
          '';
        };
      });

      overlays.default = final: prev: {
        protoc-gen-go-errors = self.packages.${final.system}.protoc-gen-go-errors;
      };
    };
}