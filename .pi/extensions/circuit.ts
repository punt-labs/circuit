import { execFile } from "node:child_process";
import { promisify } from "node:util";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

const execFileAsync = promisify(execFile);

export default function circuitExtension(pi: ExtensionAPI) {
	pi.registerCommand("circuit-validate", {
		description: "Validate a circuit playbook file",
		handler: async (args, ctx) => {
			const file = args.trim() || "examples/pr-watch.yaml";
			try {
				const result = await execFileAsync("go", ["run", "./cmd/circuit", "validate", file], {
					cwd: process.cwd(),
				});
				ctx.ui.notify(result.stdout.trim() || "circuit validation passed", "info");
			} catch (error) {
				if (isExecError(error)) {
					const message = error.stderr.trim() || error.stdout.trim() || error.message;
					ctx.ui.notify(message, "error");
					return;
				}
				ctx.ui.notify(String(error), "error");
			}
		},
	});
}

function isExecError(error: unknown): error is Error & { stdout: string; stderr: string } {
	return error instanceof Error && "stdout" in error && "stderr" in error;
}
