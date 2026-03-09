class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.13.0.tar.gz"
  sha256 "1d23054ad260f2f1adc8fb854526bd2301773eb0d4cb9bd7888534124d657129"
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
