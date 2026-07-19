{
  description = "Development environment for the AtomGit CLI";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }:
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
          pkgs = import nixpkgs { inherit system; };
          version = "0.6.0";
          commit = self.shortRev or self.dirtyShortRev or "unknown";
          buildDate =
            let
              d = self.lastModifiedDate or "19700101000000";
            in
            if builtins.stringLength d >= 14
            then "${builtins.substring 0 4 d}-${builtins.substring 4 2 d}-${builtins.substring 6 2 d}T${builtins.substring 8 2 d}:${builtins.substring 10 2 d}:${builtins.substring 12 2 d}Z"
            else "unknown";
        in
        {
          ag = pkgs.buildGoModule {
            pname = "ag";
            inherit version;
            src = self;
            vendorHash = "sha256-7K17JaXFsjf163g5PXCb5ng2gYdotnZ2IDKk8KFjNj0=";
            subPackages = [ "cmd/ag" ];
            ldflags = [
              "-s"
              "-w"
              "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Version=v${version}"
              "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Commit=${commit}"
              "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.BuildDate=${buildDate}"
            ];
          };
          default = self.packages.${system}.ag;
        });

      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
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
            ];

            CGO_ENABLED = "0";
          };
        });
    };
}
