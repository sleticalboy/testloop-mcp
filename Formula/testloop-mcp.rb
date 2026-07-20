class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.14"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.14/testloop-mcp_v0.5.14_darwin_arm64.tar.gz"
      sha256 "112c26ff0bc02943fa66a9270478d0713d324debc6fdb272502e11a860d4eaed"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.14/testloop-mcp_v0.5.14_linux_amd64.tar.gz"
      sha256 "2298e05eda8184a7bdd52973d0b4168ae0d4a1db90361c9cd48067483cdb6738"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.14/testloop-mcp_v0.5.14_linux_arm64.tar.gz"
      sha256 "0aace22cb765319ed7e245519f3ca55c5fbe07a61872a89430d4badd792f9ddb"
    end
  end

  def install
    bin.install "testloop-mcp"
    bin.install "testloop-testgen"
  end

  test do
    assert_match "Usage of testloop-mcp", shell_output("#{bin}/testloop-mcp --help 2>&1")
    assert_match "Usage: testgen", shell_output("#{bin}/testloop-testgen --help 2>&1")
  end
end
