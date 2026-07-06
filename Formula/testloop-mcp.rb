class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.4/testloop-mcp_v0.4.4_darwin_arm64.tar.gz"
      sha256 "a73295e9cff0e28ea0b567aacf0623d4cec71c21da559d89190f5eecd596ba76"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.4/testloop-mcp_v0.4.4_linux_amd64.tar.gz"
      sha256 "2f5821271557bc603c0ca85b0a4d74d6c1551549ca29139ee5b90c44adf800cc"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.4/testloop-mcp_v0.4.4_linux_arm64.tar.gz"
      sha256 "6b319850a433c6a7ff89ff36d0b0e521020f67e5852aef7bcc5c4cf9c098480c"
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
