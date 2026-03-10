class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.13.3.tar.gz"
  sha256 "14bc13a5f82ebd577e6c60f11a50f6fee165f86b323efbbc24074260c0dc1924"
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
