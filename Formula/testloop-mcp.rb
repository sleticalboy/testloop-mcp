class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.6"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.6/testloop-mcp_v0.5.6_darwin_arm64.tar.gz"
      sha256 "9561f2cf1b0440f0e93d85b3796e5b6a876a2691385da93b408002366d704d58"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.6/testloop-mcp_v0.5.6_linux_amd64.tar.gz"
      sha256 "90e85dcc4b4ce74f3e7fabaf18ed39b356c67cac20a3c2c49d2cf9e81620a766"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.6/testloop-mcp_v0.5.6_linux_arm64.tar.gz"
      sha256 "9d26dc21a071cf9e770bb79a308c8b9a55c0a2ba48630c5a3d54e316e26c7a97"
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
