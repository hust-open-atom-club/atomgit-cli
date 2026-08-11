{
  pkgs,
  mkAg,
  self,
}:

let
  commit = self.rev or self.dirtyRev or "unknown";
  version = self.shortRev or self.dirtyShortRev or "dev";
  buildDate =
    let
      d = self.lastModifiedDate or "19700101000000";
    in
    if builtins.stringLength d >= 14
    then "${builtins.substring 0 4 d}-${builtins.substring 4 2 d}-${builtins.substring 6 2 d}T${builtins.substring 8 2 d}:${builtins.substring 10 2 d}:${builtins.substring 12 2 d}Z"
    else "unknown";
in
pkgs.buildGoModule (
  mkAg {
    inherit commit buildDate;
    buildVersion = version;
  }
  // {
    inherit version;
    src = self;
    vendorHash = "sha256-4RF6GG+F/3jPx81t4bBhsKytqD6VU3o32xZPOluKy6g=";
  }
)
