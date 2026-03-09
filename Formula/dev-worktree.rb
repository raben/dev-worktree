class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.13.2.tar.gz"
  sha256 "ae2a54bcd892afe94fb626d350c0679395353964ef6585bb33f1fa1010f97c3e"
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
