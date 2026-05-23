{
  description = "A simple TLS pass-through reverse proxy based on peeking SNI messages";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-25.11";
    gomod2nix = {
      url = "github:nix-community/gomod2nix/v1.7.0";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, gomod2nix }:
    let
      version = builtins.substring 0 8 self.lastModifiedDate;
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system:
        import nixpkgs {
          inherit system;
          overlays = [
            gomod2nix.overlays.default
          ];
        }
      );
    in {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in {
          snip = pkgs.buildGoApplication {
            pname = "snip";
            inherit version;
            src = ./.;
            modules = ./gomod2nix.toml;
          };
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gopls
              gotools
              go-tools
            ];
          };
        });

      defaultPackage = forAllSystems (system: self.packages.${system}.snip);
    };
}
