class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/sleticalboy/testloop-mcp"
  version "0.5.5"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.5/testloop-mcp_v0.5.5_darwin_arm64.tar.gz"
      sha256 "66d24bc76fd58c16b82bf54d000ae16f40437da9bbc82f14c758aff795520725"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.5/testloop-mcp_v0.5.5_linux_amd64.tar.gz"
      sha256 "111e2236441cc5cfaae7e62c81dda626c751641d1f3a38a30ab82fa314c2177a"
    end

    on_arm do
      url "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.5/testloop-mcp_v0.5.5_linux_arm64.tar.gz"
      sha256 "d862746039eb89da2b242f3cb7dc0d03f6b9da88b3d66430011280be2b50c76c"
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
