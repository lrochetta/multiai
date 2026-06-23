# typed: false
# frozen_string_literal: true

# Homebrew formula for multiai
class Multiai < Formula
  desc "Route multiple AI CLIs (Claude Code, Codex, OpenCode) with isolated env profiles"
  homepage "https://rochetta.fr"
  url "https://github.com/lrochetta/multiai/archive/refs/tags/v0.5.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256" # Will be filled by goreleaser
  license "MIT"
  head "https://github.com/lrochetta/multiai.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", "-ldflags", "-s -w -X main.version=#{version}",
           "-o", bin/"multiai", "./cmd/multiai/"

    # Install shell completions
    generate_completions
  end

  def generate_completions
    output = Utils.safe_popen_read(bin/"multiai", "completion", "bash")
    (bash_completion/"multiai").write output

    output = Utils.safe_popen_read(bin/"multiai", "completion", "zsh")
    (zsh_completion/"_multiai").write output

    output = Utils.safe_popen_read(bin/"multiai", "completion", "fish")
    (fish_completion/"multiai.fish").write output
  end

  test do
    assert_match "multiai", shell_output("#{bin}/multiai version")
  end
end
