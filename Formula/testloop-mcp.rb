class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.17"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.17/testloop-mcp_v0.5.17_darwin_arm64.tar.gz"
      sha256 "1856118a85a37519b42c0f8bfa74cb228c03a8c0a1c8dc3a95ca4e0e52b6f581"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.17/testloop-mcp_v0.5.17_linux_amd64.tar.gz"
      sha256 "780d77b8a62987261455eace9d3576bd63c3e6e0f6ea9b6bd886494eb16c2118"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.17/testloop-mcp_v0.5.17_linux_arm64.tar.gz"
      sha256 "fabd308d1b383c62a186b978c780159c0a6b66a7124d6d7e9520d38516f427b2"
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
