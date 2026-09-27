{
  description = "Vanity import paths for Go modulesr";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = {nixpkgs, ...}: let
    systems = ["x86_64-linux" "aarch64-linux" "aarch64-darwin"];
    forAll = f: nixpkgs.lib.genAttrs systems (s: f nixpkgs.legacyPackages.${s});
  in {
    devShells = forAll (pkgs: {
      default = pkgs.mkShell {
        packages = [pkgs.go pkgs.tinygo pkgs.wrangler];
        CGO_ENABLED = "0";
      };
    });
    formatter = forAll (pkgs: pkgs.alejandra);
  };
}
