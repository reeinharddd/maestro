import { tool } from "@opencode-ai/plugin"

export default tool({
  description: "Check Definition of Done compliance for current changes. Run before commit.",
  args: {},
  async execute(_args, context) {
    const { execSync } = await import("child_process")
    const issues: string[] = []
    const passed: string[] = []
    let exitCode = 0

    const runCheck = (cmd: string, label: string, advisory = false) => {
      try {
        execSync(cmd, { cwd: context.worktree, stdio: "pipe", timeout: 60000 })
        passed.push(`✅ ${label}`)
      } catch (e: any) {
        const msg = `❌ ${label}: ${e.stderr?.toString().slice(0, 200) || e.message}`
        if (advisory) {
          issues.push(`⚠️  ${label} (advisory)`)
        } else {
          issues.push(msg)
          exitCode = 1
        }
      }
    }

    runCheck("go build ./...", "build passes")
    runCheck("go vet ./...", "vet passes")
    runCheck("go test -race ./...", "tests pass")

    // Check for TODOs in staged changes
    try {
      const todos = execSync("git diff --cached -G'TODO|FIXME|HACK|XXX' --name-only", { cwd: context.worktree, encoding: "utf8" }).trim()
      if (todos) {
        issues.push(`⚠️  TODOs/FIXMEs in staged files:\n${todos}`)
      } else {
        passed.push("✅ no TODOs added")
      }
    } catch { /* no staged changes */ }

    // Check coverage
    try {
      execSync("go test -coverprofile=coverage.out ./...", { cwd: context.worktree, stdio: "pipe", timeout: 120000 })
      const out = execSync("go tool cover -func=coverage.out", { cwd: context.worktree, encoding: "utf8" })
      const totalLine = out.split("\n").find(l => l.includes("total:"))
      if (totalLine) passed.push(`📊 ${totalLine.trim()}`)
    } catch { /* coverage advisory */ }

    return {
      exitCode,
      summary: exitCode === 0 ? "✅ DoD passed" : "❌ DoD failed",
      details: [...passed, ...issues].join("\n"),
    }
  },
})
