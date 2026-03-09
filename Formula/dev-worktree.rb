class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.12.1.tar.gz"
  sha256 "793b68715269d2c419a1229a931c802a5fe060a839beb0c1c479179c20c4ecc1"
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
