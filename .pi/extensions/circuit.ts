import { execFile } from "node:child_process";
import { readdir } from "node:fs/promises";
import { basename, join } from "node:path";
import { promisify } from "node:util";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { parseCircuitCommand } from "./parse.js";

const execFileAsync = promisify(execFile);
const MACHINE_DIR = "machines";

export default function circuitExtension(pi: ExtensionAPI) {
	pi.registerCommand("circuit", {
		description: "Manage the active Circuit machine: /circuit <list|start|status|advance|stop>",
		handler: async (args, ctx) => {
			const parsed = parseCircuitCommand(args);
			switch (parsed.verb) {
				case "list": {
					const machines = await listMachines();
					ctx.ui.notify(machines.length > 0 ? machines.join("\n") : "No machines found", "info");
					return;
				}
				case "start": {
					if (!parsed.argument) {
						ctx.ui.notify("Usage: /circuit start <machine>", "warning");
						return;
					}
					const result = await runCircuit(["start", parsed.argument]);
					if (!result.ok) {
						ctx.ui.notify(result.message, "error");
						return;
					}
					ctx.ui.notify(result.message, "info");
					return;
				}
				case "status": {
					const result = await runCircuit(["status"]);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
				case "advance": {
					if (!parsed.argument) {
						ctx.ui.notify("Usage: /circuit advance <event>", "warning");
						return;
					}
					const result = await runCircuit(["advance", parsed.argument]);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
				case "stop": {
					const result = await runCircuit(["stop"]);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
			}
		},
	});
}

async function listMachines(): Promise<string[]> {
	const entries = await readdir(join(process.cwd(), MACHINE_DIR));
	return entries
		.filter((entry) => entry.endsWith(".mch"))
		.map((entry) => basename(entry, ".mch"))
		.sort();
}

interface CircuitResult {
	ok: boolean;
	message: string;
}

async function runCircuit(args: string[]): Promise<CircuitResult> {
	try {
		const result = await execFileAsync("go", ["run", "./cmd/circuit", ...args], {
			cwd: process.cwd(),
		});
		return { ok: true, message: result.stdout.trim() || "circuit command completed" };
	} catch (error) {
		if (isExecError(error)) {
			return { ok: false, message: error.stderr.trim() || error.stdout.trim() || error.message };
		}
		return { ok: false, message: String(error) };
	}
}

function isExecError(error: unknown): error is Error & { stdout: string; stderr: string } {
	return error instanceof Error && "stdout" in error && "stderr" in error;
}
