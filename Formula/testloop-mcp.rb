class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.11"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.11/testloop-mcp_v0.4.11_darwin_arm64.tar.gz"
      sha256 "34d883962b35e4980de2b66d90b97a7c09f81ee5e532455cd6ce89546bc3cff4"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.11/testloop-mcp_v0.4.11_linux_amd64.tar.gz"
      sha256 "a8657b2da9246c4ce2c5fb572f4dc1ce316ce33491d41f01be2ca1cb724f84a7"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.11/testloop-mcp_v0.4.11_linux_arm64.tar.gz"
      sha256 "1d4a91d8433fd12d85c188705c1d1f8fcee30c038968253b4bae129d6db4a7d1"
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
