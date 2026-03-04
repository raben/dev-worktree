class WtDev < Formula
  desc "Isolated parallel development environments with worktree + devcontainer"
  homepage "https://github.com/raben/wt-dev"
  head "https://github.com/raben/wt-dev.git", branch: "main"
  license "MIT"

  depends_on "jq"

  def install
    bin.install Dir["bin/*"]
  end

  test do
    system "#{bin}/dev", "--version"
  end
end
