class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.15"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.15/testloop-mcp_v0.5.15_darwin_arm64.tar.gz"
      sha256 "afbb7a13ba54be927109592413ed30f5c290341b4168e21d1ace33aa1312f228"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.15/testloop-mcp_v0.5.15_linux_amd64.tar.gz"
      sha256 "0ce3d0ac41ac7f21088d11d255842ac058463293014e046ed0b6efe630b200c2"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.15/testloop-mcp_v0.5.15_linux_arm64.tar.gz"
      sha256 "d3ef345627e1c6a8b2709fcdb2eb7e147938b471abd79b85a7e0adbef3132fe3"
    end
  end

  def install
    bin.install "testloop-mcp"
    bin.install "testloop-testgen"
  end

  test do
    assert_match "Usage of testloop-mcp", shell_output("#{bin}/testloop-mcp --help 2>&1")
    assert_match "Usage: testgen", shell_output("#{bin}/testloop-testgen --help 2>&1")
  end
end
