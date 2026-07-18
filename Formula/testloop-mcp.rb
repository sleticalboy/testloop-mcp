class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.4/testloop-mcp_v0.5.4_darwin_arm64.tar.gz"
      sha256 "11be06e62df3ab825b714e172bb7121a9f061b77774d5130cede6bb9dc1b5f5f"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.4/testloop-mcp_v0.5.4_linux_amd64.tar.gz"
      sha256 "918f6e6a22f57ff0429e6daeae8af05f20af7b32374c92bfbb322a3cb0396841"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.4/testloop-mcp_v0.5.4_linux_arm64.tar.gz"
      sha256 "3e98b63765e204b6d80319ed75797d0933f818399af336c894bf526c92cac1a7"
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
