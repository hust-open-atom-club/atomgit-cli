{
  pkgs,
  mkAg,
}:

let
  version = "unstable-baadf0a89a49b17c07f4f1bbfa37ef2da059b435";
  commit = builtins.substring 9 40 version;
in
pkgs.buildGoModule (
  mkAg {
    inherit commit;
    buildVersion = builtins.substring 0 7 commit;
  }
  // {
    inherit version;
    vendorHash = "sha256-iv1FSZyFaG2IL7DzI8UZp77OjfbQG6A4LGSR/TQFDQI=";
    src = pkgs.fetchzip {
      url = "https://raw.atomgit.com/hust-open-atom-club/atomgit-cli/archive/refs/heads/${commit}.tar.gz";
      hash = "sha256-7wPLSW/3m1LSsu+i4fCJBZ1NS/OATbN5Np0nVlfrk/A=";
    };
  }
)
