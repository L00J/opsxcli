# OpsXCLI Homebrew Formula (参考模板)
# 安装方式:
#   brew tap L00J/tap
#   brew install opsxcli
#
# ⚠️ 此文件仅供参考！实际 Formula 由 GoReleaser 自动生成并推送到 L00J/homebrew-tap
# GoReleaser 会根据 .goreleaser.yaml 中的 brews 配置自动生成正确的 Formula
# （包含按平台选择二进制 URL + 正确的 sha256）
#
# 防代码泄露策略:
#   - Formula url 指向 GitHub Release 的预编译 tar.gz（二进制）
#   - 不放源码 URL — AI/爬虫只能下载到编译后的二进制
#   - 源码仅在 Gitee 私有仓库

class Opsxcli < Formula
  desc "DevOps CLI toolkit with 70+ commands, AI agent, TUI dashboard"
  homepage "https://gitee.com/opsx-tools/opsxcli"
  # GoReleaser 自动填充: 版本号 + 二进制下载 URL + sha256
  url "https://github.com/L00J/opsxcli/releases/download/VERSION_PLACEHOLDER/opsxcli_VERSION_Darwin_arm64.tar.gz"
  sha256 "SHA256_PLACEHOLDER"
  version "VERSION_PLACEHOLDER"
  license "MIT"

  def install
    bin.install "opsxcli"
    generate_completions_from_executable(bin/"opsxcli", "completion")
  end

  test do
    assert_match "opsxcli", shell_output("#{bin}/opsxcli version")
  end
end
