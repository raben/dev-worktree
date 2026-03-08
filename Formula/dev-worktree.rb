class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.9.9.tar.gz"
  sha256 "5e1e4ace29d1b3816421c8811ec03faead66ca22e44951c9058ccf2b076b879f"
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
