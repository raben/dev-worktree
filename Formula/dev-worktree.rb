class DevWorktree < Formula
  desc "Isolated parallel development environments with worktree + Docker"
  homepage "https://github.com/raben/dev-worktree"
  url "https://github.com/raben/dev-worktree/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/raben/dev-worktree.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/autor-dev/dev-worktree/cmd.version=#{version}"
    system "go", "build", *std_go_args(ldflags:), "-o", bin/"dev"
  end

  test do
    system "#{bin}/dev", "--version"
  end
end
