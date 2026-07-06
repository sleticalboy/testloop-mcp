class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.3/testloop-mcp_v0.4.3_darwin_arm64.tar.gz"
      sha256 "e0f54687a34d3f783c9268976da5778beefeb6d709c55a649b3a2a351392a48b"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.3/testloop-mcp_v0.4.3_linux_amd64.tar.gz"
      sha256 "f700a493489f326ef889fcd16ef757e3c00b0a861c5018cb94f167d510fa9b6d"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.3/testloop-mcp_v0.4.3_linux_arm64.tar.gz"
      sha256 "817b5e24768a789e5c634e28fd0556e9d1381dce77846e017085d43319057ef0"
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
