class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.10"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.10/testloop-mcp_v0.5.10_darwin_arm64.tar.gz"
      sha256 "ce9dab43efc95bd13d6dbbcfd9e36f268ab684839c4135975600dccd89c94bc7"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.10/testloop-mcp_v0.5.10_linux_amd64.tar.gz"
      sha256 "b5e8ca76b25d0b3a72be54a5baf4dd68376f387358c58011cd75ccfc9ca910d7"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.10/testloop-mcp_v0.5.10_linux_arm64.tar.gz"
      sha256 "6dd4bcb5937c07fb7a0c095629f5e2dfdf2ed1c46cc565816c762f79ec5060f8"
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
