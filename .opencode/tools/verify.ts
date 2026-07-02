import { tool } from "@opencode-ai/plugin"

export default tool({
  description: "Run full verification: build + vet + test-race + coverage. Call before every commit.",
  args: {
    scope: tool.schema.string().optional().describe("Package scope to test (e.g. './internal/db/...'). Default: all"),
  },
  async execute(args, context) {
    const { execSync } = await import("child_process")
    const results: string[] = []
    let exitCode = 0

    const run = (cmd: string, label: string) => {
      try {
        execSync(cmd, { cwd: context.worktree, stdio: "pipe", timeout: 120000 })
        results.push(`✅ ${label}`)
      } catch (e: any) {
        results.push(`❌ ${label}\n${e.stderr?.toString() || e.message}`)
        exitCode = 1
      }
    }

    run("go build ./...", "build")
    run("go vet ./...", "vet")

    if (args.scope) {
      run(`go test -race -coverprofile=coverage.out ${args.scope}`, `test ${args.scope}`)
    } else {
      run("go test -race -coverprofile=coverage.out ./...", "test all")
    }

    try {
      const { execSync } = await import("child_process")
      const out = execSync("go tool cover -func=coverage.out", { cwd: context.worktree, encoding: "utf8" })
      results.push("\n--- Coverage ---")
      const lines = out.split("\n").slice(-10)
      results.push(...lines.filter(l => l.includes("total:")))
    } catch {
      // coverage check is advisory
    }

    return { exitCode, results: results.join("\n") }
  },
})
