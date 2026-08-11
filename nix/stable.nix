{
  pkgs,
  mkAg,
}:

let
  version = "0.7.1";
  commit = "v${version}";
in
pkgs.buildGoModule (
  mkAg {
    inherit commit;
    buildVersion = "v${version}";
  }
  // {
    inherit version;
    vendorHash = "sha256-4RF6GG+F/3jPx81t4bBhsKytqD6VU3o32xZPOluKy6g=";
    src = pkgs.fetchzip {
      url = "https://raw.atomgit.com/hust-open-atom-club/atomgit-cli/archive/refs/heads/v${version}.tar.gz";
      hash = "sha256-SyOGkxDgkfQUeaHp6F+FhTLqWYCWvG7RVqVw1wfE1Uw=";
    };
  }
)
