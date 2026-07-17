class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.1/testloop-mcp_v0.5.1_darwin_arm64.tar.gz"
      sha256 "0980d7949c987c69dfdac606e9a143fabb7770e00daaa86db7b3e865a49bda16"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.1/testloop-mcp_v0.5.1_linux_amd64.tar.gz"
      sha256 "923f9a5d17d0e0bb4e8a90e8740fa2bf308f708efd2aedaef02fb5a4c58753fc"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.1/testloop-mcp_v0.5.1_linux_arm64.tar.gz"
      sha256 "8e94231b15405d07fb65c321e9710b364378dadde02bea88bafc64cb6f5b2f9b"
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
