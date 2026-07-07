class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.4.10"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.10/testloop-mcp_v0.4.10_darwin_arm64.tar.gz"
      sha256 "e0aff24ebe4c3e2cfa71998c950f6c087aeab251e55d9984e6cec4770a8c9dc6"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.10/testloop-mcp_v0.4.10_linux_amd64.tar.gz"
      sha256 "b5ff6403f645b477939a2e4bad0d74cfb74a80a86fbee51e099cb1105dd334d9"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.10/testloop-mcp_v0.4.10_linux_arm64.tar.gz"
      sha256 "4bdf4fd898cb278d0161980d4c76e41fa7f96cff530815240b887b638b3535c4"
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
