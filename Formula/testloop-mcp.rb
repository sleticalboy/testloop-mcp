class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.3/testloop-mcp_v0.5.3_darwin_arm64.tar.gz"
      sha256 "7e127e43123a7e8557bf99b50d91a96f6db6b436113b45993eb823551e227cb2"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.3/testloop-mcp_v0.5.3_linux_amd64.tar.gz"
      sha256 "4895c0d81c7a2e45bb25c38abcf36f7b387caad723ddf54f379aa69c99743d90"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.3/testloop-mcp_v0.5.3_linux_arm64.tar.gz"
      sha256 "fec4afa1e3547f064886a61e53bf5a6df4883144d0e27374266636b083ea06f0"
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
