class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.11.0.tar.gz"
  sha256 "dcd28e7da7fb0c8460b7b76b716a2b91ddf39202241bc3f328fea00c2c9021c4"
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
