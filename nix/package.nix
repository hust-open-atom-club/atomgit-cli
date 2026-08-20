{
  pkgs,
}:
{
  buildVersion,
  commit,
  buildDate ? "unknown",
}:
{
  pname = "ag";
  subPackages = [ "cmd/ag" ];
  proxyVendor = true;
  env.GOPROXY = "https://goproxy.cn|https://mirrors.aliyun.com/goproxy/|direct";
  ldflags = [
    "-s"
    "-w"
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Version=${buildVersion}"
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Commit=${commit}"
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.BuildDate=${buildDate}"
    # Keep old stable sources on the package-manager update policy until stable
    # points to a release that no longer defines distribution metadata.
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Source=nix"
  ];
}
