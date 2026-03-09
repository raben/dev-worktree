class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.13.1.tar.gz"
  sha256 "e5673fd1996bf0729a22c0fe095fdb50e44662128a400bceb40644747a5e7dcf"
  license "MIT"
  head "https://github.com/raben/dev-worktree.git", branch: "main"

  depends_on "fzf"
  depends_on "jq"

  def install
    bin.install Dir["bin/*"]
    (lib/"dev").install Dir["lib/dev/*"]
  end

  test do
    system "#{bin}/dev", "--version"
  end
end
