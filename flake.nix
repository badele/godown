{
  description = "godown - A simple markdown web server";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "godown";
          version = "0.1.0";
          src = ./.;

          vendorHash = "sha256-nlaO32vKmi3QVp9rZ8UCn5LIfBhLlkkiYMvuRVRK+BQ=";

          meta = with pkgs.lib; {
            description = "A simple Markdown file server written in Go";
            homepage = "https://github.com/badele/godown";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go development
            go
            gopls
            gotools # goimports, godoc, etc.
            go-tools # staticcheck, etc.

            # Build tools
            just

            # Pre-commit hooks
            pre-commit

            # Docker linting
            hadolint
          ];

          shellHook = ''
            echo "🚀 godown development environment"
            echo "Go version: $(go version)"
            echo ""
            just
          '';
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/godown";
        };
      }
    );
}
