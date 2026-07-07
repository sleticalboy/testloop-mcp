class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.9"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.9/testloop-mcp_v0.4.9_darwin_arm64.tar.gz"
      sha256 "e35e65c6ec8298d029dd6688c61b602db12febfee5f9fecc7d28dfca878437be"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.9/testloop-mcp_v0.4.9_linux_amd64.tar.gz"
      sha256 "791a71fe5eb74846291fe1873db8e867ecfb7ec814c938c3c10d04955ce602ca"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.9/testloop-mcp_v0.4.9_linux_arm64.tar.gz"
      sha256 "73c0fea13b8bb78113230a7629fc9941b739c4edaa2d610704e0a008398ab676"
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
