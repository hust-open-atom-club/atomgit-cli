{
  pkgs,
  mkAg,
}:

let
  version = "0.7.5";
  commit = "0bd03bcedbe8db803255cd7ec8a2b367a38eefa2";
  buildDate = "2026-11-27T13:02:09+08:00";
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
      hash = "sha256-StG1smozkl7hLypXoMtaONtzkkPk+0y5D7ZwLzyr0B8=";
    };
  }
)
