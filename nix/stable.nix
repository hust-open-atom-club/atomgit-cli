{
  pkgs,
  mkAg,
}:

let
  version = "0.7.0";
  commit = "v${version}";
in
pkgs.buildGoModule (
  mkAg {
    inherit commit;
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
