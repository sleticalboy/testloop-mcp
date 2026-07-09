class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.12"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.12/testloop-mcp_v0.4.12_darwin_arm64.tar.gz"
      sha256 "cc3b3ed1cf7feddae64119d388f399746258b08972c79e774d5f2bf7b7f6261a"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.12/testloop-mcp_v0.4.12_linux_amd64.tar.gz"
      sha256 "5c95f66ed7236d20e046be6be7b7edbf7b357eb13326a00132310cb1126f09c5"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.12/testloop-mcp_v0.4.12_linux_arm64.tar.gz"
      sha256 "de0a54a674a290e86fe300f16015c1e456a7b907bd767f00bb668575e5dee4fd"
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
