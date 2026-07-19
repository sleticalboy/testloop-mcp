class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.9"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.9/testloop-mcp_v0.5.9_darwin_arm64.tar.gz"
      sha256 "0c2b78ab0fdaafbb18e86f4c3a1d9e52b774805b8076f1d0afb3e9c7de405be4"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.9/testloop-mcp_v0.5.9_linux_amd64.tar.gz"
      sha256 "ec2ad7b1961b691aa55161af77ca0b501d417eb970532578643e1da70c81affc"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.9/testloop-mcp_v0.5.9_linux_arm64.tar.gz"
      sha256 "1d787632369faef3193df510fdd52bb05c49931cca9435f2cb4cfca969c96411"
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
