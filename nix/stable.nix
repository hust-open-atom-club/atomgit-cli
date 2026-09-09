{
  pkgs,
  mkAg,
}:

let
  version = "0.7.3";
  commit = "6cfffd1f9ffc8e240baff316031f89835cb013e8";
  buildDate = "2026-09-05T17:05:40+08:00";
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
      hash = "sha256-Yvv65+tNO9JJ4aW0KGDbjYzXAEWp9QFzCrrYMwdSl8M=";
    };
  }
)
