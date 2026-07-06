class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.7"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.7/testloop-mcp_v0.4.7_darwin_arm64.tar.gz"
      sha256 "71a1199ef6c9bc3bea35ec05a12f151fab73dfe62547d039b1a2a9e62d7d7a06"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.7/testloop-mcp_v0.4.7_linux_amd64.tar.gz"
      sha256 "731376d6d0c3d593cf84a8973e1f6a180c049b558182e735339efbc530febc34"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.7/testloop-mcp_v0.4.7_linux_arm64.tar.gz"
      sha256 "706bc8f69fef01e8bc2a7a97ea8dc90bc809821b372df9bfe7f8917465554174"
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
