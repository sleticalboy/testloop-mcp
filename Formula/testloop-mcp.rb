class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.20"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.20/testloop-mcp_v0.5.20_darwin_arm64.tar.gz"
      sha256 "a6d5a85846bdfc5d45c69988ca98d909ef4506dce1a5f93cfa8a4f5900b40ea7"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.20/testloop-mcp_v0.5.20_linux_amd64.tar.gz"
      sha256 "693b5f68aa98e7712f21a92adcda4e931c6eb975cc210e189ea353645548a086"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.20/testloop-mcp_v0.5.20_linux_arm64.tar.gz"
      sha256 "e230011d70ed2f1665279198438d69c184460c6ff799d33d5d632d91f9bb255c"
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
