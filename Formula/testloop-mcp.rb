class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.8"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.8/testloop-mcp_v0.4.8_darwin_arm64.tar.gz"
      sha256 "6254ef44cc8894957127f3eccc9856796b9c6895cf863ace8087491e78f58f11"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.8/testloop-mcp_v0.4.8_linux_amd64.tar.gz"
      sha256 "ee5ecf703e8989a04b3126922b36b6af5c35031fc73bb018c5150f8c3e1af207"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.8/testloop-mcp_v0.4.8_linux_arm64.tar.gz"
      sha256 "a271cf0a4c4a1406df86ac8f69d0939b6b48697c366efbcb47347b2bb609d8cf"
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
