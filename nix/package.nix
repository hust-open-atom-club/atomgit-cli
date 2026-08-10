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
  env.GOPROXY = "https://goproxy.cn,direct";
  ldflags = [
    "-s"
    "-w"
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Version=${buildVersion}"
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Commit=${commit}"
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.BuildDate=${buildDate}"
    "-X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Source=nix"
  ];
}
