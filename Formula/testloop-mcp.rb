class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.12"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.12/testloop-mcp_v0.5.12_darwin_arm64.tar.gz"
      sha256 "b05e33fd909e94e8acc9e565dad7a99129088d30306ce85eab20eaf359fa94ac"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.12/testloop-mcp_v0.5.12_linux_amd64.tar.gz"
      sha256 "94a8d44d65f218392bf9b20662268fd0943b16c56b2ca5a9230fac98ae63e664"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.12/testloop-mcp_v0.5.12_linux_arm64.tar.gz"
      sha256 "72070c95fc4220bd8396a8abed984f1d8200b36245a99d50763549088785fe6b"
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
