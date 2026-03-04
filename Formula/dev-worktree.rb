class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v0.6.0.tar.gz"
  sha256 "f83f3cbfa212a43a220cabdc79077b909e27db1c29bee766d5f2568cdf1b9535"
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
