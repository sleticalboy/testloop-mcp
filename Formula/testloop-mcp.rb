class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.2/testloop-mcp_v0.5.2_darwin_arm64.tar.gz"
      sha256 "d4653b71a2f03224475232471e89f5736199cdade3e5a0f1dfef664c5f62d634"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.2/testloop-mcp_v0.5.2_linux_amd64.tar.gz"
      sha256 "7fa25b86a1b73a3e95ba3b4a731e841d78f72a5642dd97fe9b104d030d103867"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.2/testloop-mcp_v0.5.2_linux_arm64.tar.gz"
      sha256 "90bb0fc07457f50d659dcff9a2b68d79333b192f84004b7733b7b23a95df72cc"
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
