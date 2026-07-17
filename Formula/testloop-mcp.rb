class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.0/testloop-mcp_v0.5.0_darwin_arm64.tar.gz"
      sha256 "3f3e8bc9a29a61797a86aa286f7a77d3c594157989d958358af8009c9a2e6849"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.0/testloop-mcp_v0.5.0_linux_amd64.tar.gz"
      sha256 "d30de31461f1d2b1136da700d98cb69228230d5869c696c7c1f32a25bde517b7"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.0/testloop-mcp_v0.5.0_linux_arm64.tar.gz"
      sha256 "279c34702a219876569df175bcbf8370abae51d666cd05e622bf1db6ee415e8c"
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
