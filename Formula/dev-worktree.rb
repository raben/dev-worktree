class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.3.0.tar.gz"
  sha256 "479405990536bd09278b4ef7b28a05f6ae1c79928e3ff0af3b72fcd6782d6b03"
  license "MIT"
  head "https://github.com/raben/dev-worktree.git", branch: "main"

  depends_on "jq"

  def install
    bin.install Dir["bin/*"]
  end

  test do
    system "#{bin}/dev", "--version"
  end
end
