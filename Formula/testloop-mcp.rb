class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.11"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.11/testloop-mcp_v0.5.11_darwin_arm64.tar.gz"
      sha256 "9ae78cb80bb6a34b60c8d92c7815b4b3ae0f02110d1aafa95adb916b885bb4f1"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.11/testloop-mcp_v0.5.11_linux_amd64.tar.gz"
      sha256 "b39828ca34c063fce87bf54b4717607031629653b00c523a245c2a92b9279ea8"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.11/testloop-mcp_v0.5.11_linux_arm64.tar.gz"
      sha256 "898b20e73e3101fc79bbe64da2f3ede293f191621bf86b8e9e5f35208430744b"
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
