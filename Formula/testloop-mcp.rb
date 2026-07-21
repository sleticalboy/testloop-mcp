class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.16"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.16/testloop-mcp_v0.5.16_darwin_arm64.tar.gz"
      sha256 "842979905f206385034171b312cd4b4125750a4d4ed6c3ecec3b9f03ca0066ce"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.16/testloop-mcp_v0.5.16_linux_amd64.tar.gz"
      sha256 "a3e99c477ef0e11f01e6036989b70ef3107fd2c4968a8b2ac99d809c0cf8c40a"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.16/testloop-mcp_v0.5.16_linux_arm64.tar.gz"
      sha256 "7ff7700850f5c071478f185544d4ec3816d9f76daed1fda2606b95a6677f2313"
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
