{
  pkgs,
  mkAg,
}:

let
  version = "0.7.4";
  commit = "1b9de43514ff4f0588b06fdd4cff930fe74ce5f0";
  buildDate = "2026-10-08T20:10:44+08:00";
in
pkgs.buildGoModule (
  mkAg {
    inherit commit buildDate;
    buildVersion = "v${version}";
  }
  // {
    inherit version;
    vendorHash = "sha256-zz8W3dW+31Gi084zNjgwQ2MSQPqadFMO4+x+fbZT5eE=";
    src = pkgs.fetchzip {
      url = "https://raw.atomgit.com/hust-open-atom-club/atomgit-cli/archive/refs/heads/v${version}.tar.gz";
      hash = "sha256-8Q/LvACc3qsPwhCCYrq0LU+ji39KhjZ90qkhGM/v6AQ=";
    };
  }
)
