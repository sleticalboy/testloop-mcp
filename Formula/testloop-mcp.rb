class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.6"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.6/testloop-mcp_v0.4.6_darwin_arm64.tar.gz"
      sha256 "3a557d565a63ccf29ef3e292e7b7ff34786679222b5660344acca2a6b8ed737d"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.6/testloop-mcp_v0.4.6_linux_amd64.tar.gz"
      sha256 "3c2383f95719c8dc72deb20304807debacbd7198d198a426cc233ef4b72a2755"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.6/testloop-mcp_v0.4.6_linux_arm64.tar.gz"
      sha256 "39ef889b24532ba7b5f9be234a0f6e2e00cda3722424344cf66d2815855d098c"
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
