class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.13"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.13/testloop-mcp_v0.5.13_darwin_arm64.tar.gz"
      sha256 "02ab4bb42df2b945c0a5cf1ff1e05b75dde738d7a9d02c5bb203069de77565a9"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.13/testloop-mcp_v0.5.13_linux_amd64.tar.gz"
      sha256 "a78bcd4741f4cb57b064a0d2900af2c9ffbcf80a6a00e5808deb11ec2c0ecb5e"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.13/testloop-mcp_v0.5.13_linux_arm64.tar.gz"
      sha256 "a911d025ed9a82b65b8e1de7343728abb3e1ce27bc416ae5b681fbe5b1b10994"
    end
  end

  def install
    bin.install "testloop-mcp"
    bin.install "testloop-testgen"
  end

  test do
    assert_match "Usage of testloop-mcp", shell_output("#{bin}/testloop-mcp --help 2>&1", 2)
    assert_match "Usage: testgen", shell_output("#{bin}/testloop-testgen --help 2>&1", 2)
  end
end
