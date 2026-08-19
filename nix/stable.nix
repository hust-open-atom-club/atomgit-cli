{
  pkgs,
  mkAg,
}:

let
  version = "0.7.2";
  commit = "fd2d0c29349e2251732711e877a1a58fabbeec54";
  buildDate = "2026-08-19T00:13:48+08:00";
in
pkgs.buildGoModule (
  mkAg {
    inherit commit buildDate;
    buildVersion = "v${version}";
  }
  // {
    inherit version;
    vendorHash = "sha256-lvnlTenDlg5pTJghBZRRy8J5vig1IW9iEISN93cnRfs=";
    src = pkgs.fetchzip {
      url = "https://raw.atomgit.com/hust-open-atom-club/atomgit-cli/archive/refs/heads/v${version}.tar.gz";
      hash = "sha256-E1T093LkccgLPPNs5OokxY5tw4HEMmRy+ulaROnuSCE=";
    };
  }
)
