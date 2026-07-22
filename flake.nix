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
          stableVersion = "0.6.0";
          stableCommit = "fc5d430239fd3b420cda2fd6bb5ebe1e2e50b85d";
          stableBuildDate = "2026-07-19T00:40:02+08:00";
          stableSrcHash = "sha256-rpPVg0u4T0bZbBYHxCTZQli5JVzA2+xPjzmSebsqf5I=";
          stableVendorHash = "sha256-7K17JaXFsjf163g5PXCb5ng2gYdotnZ2IDKk8KFjNj0=";

          latestCommit = self.rev or self.dirtyRev or "unknown";
          latestVersion = self.shortRev or self.dirtyShortRev or "dev";
          latestBuildDate =
            let
              d = self.lastModifiedDate or "19700101000000";
            in
            if builtins.stringLength d >= 14
            then "${builtins.substring 0 4 d}-${builtins.substring 4 2 d}-${builtins.substring 6 2 d}T${builtins.substring 8 2 d}:${builtins.substring 10 2 d}:${builtins.substring 12 2 d}Z"
            else "unknown";
          latestVendorHash = "sha256-7K17JaXFsjf163g5PXCb5ng2gYdotnZ2IDKk8KFjNj0=";

          mkAg = {
            version,
            buildVersion ? version,
            src,
            vendorHash,
            commit,
            buildDate,
          }: pkgs.buildGoModule {
            pname = "ag";
            inherit version src vendorHash;
            subPackages = [ "cmd/ag" ];
            ldflags = [
              "-s"
              "-w"
              "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Version=${buildVersion}"
              "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Commit=${commit}"
              "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.BuildDate=${buildDate}"
            ];
          };
        in
        rec {
          stable = mkAg {
            version = stableVersion;
            buildVersion = "v${stableVersion}";
            commit = stableCommit;
            buildDate = stableBuildDate;
            vendorHash = stableVendorHash;
            src = pkgs.fetchurl {
              url = "https://raw.atomgit.com/hust-open-atom-club/atomgit-cli/archive/refs/heads/v${stableVersion}.tar.gz";
              hash = stableSrcHash;
            };
          };

          latest = mkAg {
            version = latestVersion;
            buildVersion = latestVersion;
            commit = latestCommit;
            buildDate = latestBuildDate;
            src = self;
            vendorHash = latestVendorHash;
          };

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
            ];

            CGO_ENABLED = "0";
          };
        });
    };
}
