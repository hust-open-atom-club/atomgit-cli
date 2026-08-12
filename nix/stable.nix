{
  pkgs,
  mkAg,
}:

let
  version = "0.7.1";
  commit = "783e32c68a4842d22a12882d3e11638828e101f7";
  buildDate = "2026-08-09T20:02:22+08:00";
in
pkgs.buildGoModule (
  mkAg {
    inherit commit buildDate;
    buildVersion = "v${version}";
  }
  // {
    inherit version;
    vendorHash = "sha256-iv1FSZyFaG2IL7DzI8UZp77OjfbQG6A4LGSR/TQFDQI=";
    src = pkgs.fetchzip {
      url = "https://raw.atomgit.com/hust-open-atom-club/atomgit-cli/archive/refs/heads/v${version}.tar.gz";
      hash = "sha256-SyOGkxDgkfQUeaHp6F+FhTLqWYCWvG7RVqVw1wfE1Uw=";
    };
  }
)
