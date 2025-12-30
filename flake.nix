{
  description = "A simple Go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs@{ self, nixpkgs, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      perSystem = { pkgs, ... }:
        let
          # Use custom Atlas package (official binary) to support 'ent://' schema
          atlas = import ./nix/atlas.nix { inherit pkgs; };
        in
        {
          devShells.default = pkgs.mkShell {
            buildInputs = [
              pkgs.go
              pkgs.gopls
              pkgs.delve
              pkgs.nodejs_22
              pkgs.pnpm
              pkgs.golangci-lint
              pkgs.sqlite
              pkgs.air
              atlas
            ];

            CGO_ENABLED = "1";
            CGO_CFLAGS = "-O1";
          };
        };
    };
}
