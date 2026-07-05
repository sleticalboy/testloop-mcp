class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.2/testloop-mcp_v0.4.2_darwin_arm64.tar.gz"
      sha256 "33951620deaa53ae631569c3379ce8ed96696f7d2bc96ef7106f27dc85422e9a"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.2/testloop-mcp_v0.4.2_linux_amd64.tar.gz"
      sha256 "c68fa24d10a8592f72873bf6b5fe8f581a03435acbbc8e3f5391e879d2f18a32"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.2/testloop-mcp_v0.4.2_linux_arm64.tar.gz"
      sha256 "36bc833f77c6fdb1ca6178985cc56aec1ddd64d47b2ec74e8b6787f6b5e0ba99"
    end
  end

  def install
    bin.install "testloop-mcp"
    bin.install "testloop-testgen"
  end

  test do
    system bin/"testloop-mcp", "--help"
    system bin/"testloop-testgen", "--help"
  end
end
