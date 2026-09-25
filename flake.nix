{
  description = "Gabagool Flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        default = pkgs.buildGoLatestModule {
          pname = "gabagool";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-pZRX9v4pK2rjTJzqrc9yi1Kiu4cSGg0JH6q45W1TjBM=";

          ldflags = [
            "-s"
            "-w"
          ];

        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            # Go
            go_latest
            gopls
            golangci-lint
            golangci-lint-langserver
            delve

            # Development
            git
            gnumake
          ];
        };
      });
    };
}
