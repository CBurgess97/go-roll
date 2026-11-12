{
  description = "Go development environment";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    let
      forAllSystems =
        f:
        nixpkgs.lib.genAttrs [
          "x86_64-linux"
          "aarch64-linux"
        ] (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools # goimports, etc.
            delve # debug: `dlv debug`
          ];
          hardeningDisable = [ "fortify" ]; # fortify breaks delve on cgo builds without optimisation
          GOTOOLCHAIN = "local"; # use the Go from nixpkgs; don't auto-download others
        };
      });
    };
}
