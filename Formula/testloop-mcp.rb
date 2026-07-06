class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.5"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.5/testloop-mcp_v0.4.5_darwin_arm64.tar.gz"
      sha256 "368627846fa8986a27e861a5e07c686dc87206d586cfea5ea11bbb74803f562b"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.5/testloop-mcp_v0.4.5_linux_amd64.tar.gz"
      sha256 "f0f0c237e4b56db4b46d423f7056e81c16e7ea7ce5d51a349dad3a63c4ce7f79"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.5/testloop-mcp_v0.4.5_linux_arm64.tar.gz"
      sha256 "337ad722ba4118cd6f9d51d23f3c956832095b1b9f352ab3ae3d1ada81154ac1"
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
