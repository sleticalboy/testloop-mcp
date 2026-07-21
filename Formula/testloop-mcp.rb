class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.19"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.19/testloop-mcp_v0.5.19_darwin_arm64.tar.gz"
      sha256 "a3c846e22df0e9313dc31dcbae00a453645c482ebdd39b9a1170c0493016e444"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.19/testloop-mcp_v0.5.19_linux_amd64.tar.gz"
      sha256 "236d56266d5c464d3ed93f8d63e57736353ef0321fb4c0a6bb0e4044cb6d71a4"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.19/testloop-mcp_v0.5.19_linux_arm64.tar.gz"
      sha256 "fd36a61d152f60a9a0e926cc5bdc190813c5cd676dc498d2dceb7eca2126bc11"
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
