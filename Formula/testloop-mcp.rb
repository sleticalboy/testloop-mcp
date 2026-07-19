class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.8"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.8/testloop-mcp_v0.5.8_darwin_arm64.tar.gz"
      sha256 "045b6417a6e0e92f36d61f82646d0a1bc98be1dfbc1b9c32a6eb7046e5165233"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.8/testloop-mcp_v0.5.8_linux_amd64.tar.gz"
      sha256 "963525ec3b913f4f86a4fd19e5355ce208c97a27168d869361a92d656ea66ecb"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.8/testloop-mcp_v0.5.8_linux_arm64.tar.gz"
      sha256 "58067469c710713689437f18a67d11efa4da50ca876f2b471b6f9366359e35e0"
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
