class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.7"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.7/testloop-mcp_v0.5.7_darwin_arm64.tar.gz"
      sha256 "d4a2942044fa7063abc6bfe731cfebfdca0e91e8e55ba927c151a4255e2fe464"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.7/testloop-mcp_v0.5.7_linux_amd64.tar.gz"
      sha256 "8df7f174772246049c6dc8d06213aeeaeab660b77222c8bd1a779cc50ff7046d"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.7/testloop-mcp_v0.5.7_linux_arm64.tar.gz"
      sha256 "11de7e16398e6cfe00e358a79b3e6457f9e895efb57e4891607c16bbe7d925be"
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
