# OpsXCLI Homebrew Formula
# 安装方式:
#   brew tap opsxcli/tap
#   brew install opsxcli
#
# 注意: 此文件由 goreleaser 自动更新版本号和 SHA256
# 手动安装: brew install --formula Formula/opsxcli.rb

class Opsxcli < Formula
  desc "面向运维和开发的集成化命令行工具集，70+ 运维命令"
  homepage "https://github.com/opsxcli/opsxcli"
  url "https://github.com/opsxcli/opsxcli/archive/refs/tags/v0.6.0.tar.gz"
  # goreleaser 发布时自动更新 sha256
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/opsxcli/opsxcli.git", branch: "master"

  depends_on "go" => :build

  def install
    # 静态编译，零 CGO 依赖
    system "go", "build", *std_go_args(
      ldflags: "-s -w -X main.version=#{version} -X main.buildTime=#{Time.now.iso8601}"
    ), "-trimpath", "."
  end

  test do
    # 验证版本输出
    assert_match version.to_s, shell_output("#{bin}/opsxcli version")
  end
end
