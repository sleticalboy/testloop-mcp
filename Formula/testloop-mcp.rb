class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.18"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.18/testloop-mcp_v0.5.18_darwin_arm64.tar.gz"
      sha256 "630bc0d2e1e49413d8e6d216dc2e8dd538f12db4832a09dde0f95269c1c34ee9"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.18/testloop-mcp_v0.5.18_linux_amd64.tar.gz"
      sha256 "61e44aeac3bd7913971f9369c00331f72ba95f24a3e20cb7b35556addf7955f7"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.18/testloop-mcp_v0.5.18_linux_arm64.tar.gz"
      sha256 "2eae8e81095e172300ff98882b1dc47ec4c3cf808e743d60c220007ee9efe463"
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
