{
  description = "Development environment for the AtomGit CLI";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.nixpkgs-darwin.url = "github:NixOS/nixpkgs/nixpkgs-26.05-darwin";

  outputs = { self, nixpkgs, nixpkgs-darwin, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = import
            (if system == "x86_64-darwin" then nixpkgs-darwin else nixpkgs)
            { inherit system; };
          mkAg = import ./nix/package.nix { inherit pkgs; };
          stable = import ./nix/stable.nix { inherit pkgs mkAg; };
          latest = import ./nix/latest.nix { inherit pkgs mkAg self; };
        in
        {
          inherit stable latest;
          ag = stable;
          default = stable;
        });

      devShells = forAllSystems (system:
        let
          pkgs = import
            (if system == "x86_64-darwin" then nixpkgs-darwin else nixpkgs)
            { inherit system; };
        in
        {
          default = pkgs.mkShellNoCC {
            packages = with pkgs; [
              go
              git
              gnutar
              gzip
              zip
              gnused
              xdg-utils
              nix-update
            ];

            CGO_ENABLED = "0";
          };
        });
    };
}
