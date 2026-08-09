import { execFile } from "node:child_process";
import { readdir } from "node:fs/promises";
import { basename, join } from "node:path";
import { promisify } from "node:util";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { parseCircuitCommand } from "../lib/parse.js";
import {
	formatContextInjections,
	parseAdvanceOutput,
	parseCircuitStatuses,
} from "../lib/context.js";

const execFileAsync = promisify(execFile);
const MACHINE_DIR = "machines";

export default function circuitExtension(pi: ExtensionAPI) {
	// -- Context injection: inject circuit state into every agent turn --

	pi.on("before_agent_start", async () => {
		const statusResult = await runCircuit(["status"]);
		if (!statusResult.ok) return;
		const statuses = parseCircuitStatuses(statusResult.message).filter(
			(status) => status.sessionState === "active" || status.sessionState === undefined,
		);
		if (statuses.length === 0) return;
		const injection = formatContextInjections(statuses);
		return {
			message: {
				customType: "circuit-state",
				content: injection,
				display: false,
			},
		};
	});

	// -- LLM tools: agent calls these instead of human slash commands --

	pi.registerTool({
		name: "circuit_list",
		label: "Circuit List",
		description: "List available circuit machines.",
		promptSnippet: "List available circuit machines from machines/*.mch",
		parameters: {},
		async execute() {
			const machines = await listMachines();
			const text = machines.length > 0 ? machines.join("\n") : "No machines found";
			return {
				content: [{ type: "text" as const, text }],
				details: { ok: true, count: machines.length },
			};
		},
	});

	pi.registerTool({
		name: "circuit_load",
		label: "Circuit Load",
		description: "Validate a circuit machine and its check bindings without starting a session.",
		promptSnippet: "Validate a circuit machine before starting it",
		parameters: {
			type: "object",
			properties: {
				machine: {
					type: "string",
					description: "Machine name, e.g. 'build-job', 'review-flow'",
				},
			},
			required: ["machine"],
		},
		async execute(_toolCallId, params) {
			const machine = (params as { machine: string }).machine;
			if (!machine) {
				return {
					content: [{ type: "text" as const, text: "missing machine parameter" }],
					details: { ok: false },
				};
			}
			const result = await runCircuit(["load", machine]);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok },
			};
		},
	});

	pi.registerTool({
		name: "circuit_scaffold",
		label: "Circuit Scaffold",
		description: "Generate missing BOOL check bindings and failing registry stubs for a machine.",
		promptSnippet: "Generate check stubs for a circuit machine",
		parameters: {
			type: "object",
			properties: {
				machine: {
					type: "string",
					description: "Machine name, e.g. 'review-flow'",
				},
			},
			required: ["machine"],
		},
		async execute(_toolCallId, params) {
			const machine = (params as { machine: string }).machine;
			if (!machine) {
				return {
					content: [{ type: "text" as const, text: "missing machine parameter" }],
					details: { ok: false },
				};
			}
			const result = await runCircuit(["scaffold", machine]);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok },
			};
		},
	});

	pi.registerTool({
		name: "circuit_start",
		label: "Circuit Start",
		description: "Start an active circuit from a named machine.",
		promptSnippet: "Start a circuit machine to begin a guided workflow",
		promptGuidelines: [
			"Use circuit_list to discover available machines before calling circuit_start.",
		],
		parameters: {
			type: "object",
			properties: {
				machine: {
					type: "string",
					description: "Machine name, e.g. 'build-job', 'review-flow'",
				},
			},
			required: ["machine"],
		},
		async execute(_toolCallId, params) {
			const machine = (params as { machine: string }).machine;
			if (!machine) {
				return {
					content: [{ type: "text" as const, text: "missing machine parameter" }],
					details: { ok: false },
				};
			}
			const result = await runCircuit(["start", machine]);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok },
			};
		},
	});

	pi.registerTool({
		name: "circuit_status",
		label: "Circuit Status",
		description: "Report active circuit session state, enabled and blocked operations.",
		promptSnippet: "Show current circuit session state and available transitions",
		promptGuidelines: [
			"Use circuit_status to check the current workflow state before deciding next steps.",
		],
		parameters: {
			type: "object",
			properties: {
				session: {
					type: "string",
					description: "Optional session ID, e.g. 'build-job-a3f8'",
				},
			},
		},
		async execute(_toolCallId, params) {
			const session = (params as { session?: string }).session;
			const result = await runCircuit(session ? ["status", session] : ["status"]);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok },
			};
		},
	});

	pi.registerTool({
		name: "circuit_advance",
		label: "Circuit Advance",
		description:
			"Request a state transition. The B machine validates the precondition and returns allowed or blocked.",
		promptSnippet: "Request a circuit state transition by event name",
		promptGuidelines: [
			"Use circuit_advance to request workflow progress. Do not claim a transition succeeded without calling circuit_advance first.",
			"Call circuit_status before circuit_advance if you need to check which operations are enabled.",
		],
		parameters: {
			type: "object",
			properties: {
				event: {
					type: "string",
					description: "The transition event name, e.g. 'start', 'requestReview', 'approve'",
				},
				session: {
					type: "string",
					description: "Optional session ID when multiple sessions are active",
				},
			},
			required: ["event"],
		},
		async execute(_toolCallId, params) {
			const { event, session } = params as { event: string; session?: string };
			if (!event) {
				return {
					content: [{ type: "text" as const, text: "missing event parameter" }],
					details: { ok: false },
				};
			}
			const result = await runCircuit(session ? ["advance", event, session] : ["advance", event]);
			const parsed = parseAdvanceOutput(result.message);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok, allowed: parsed.allowed, from: parsed.from, to: parsed.to },
			};
		},
	});

	pi.registerTool({
		name: "circuit_stop",
		label: "Circuit Stop",
		description: "Clear an active circuit session.",
		promptSnippet: "Stop an active circuit session",
		parameters: {
			type: "object",
			properties: {
				session: {
					type: "string",
					description: "Optional session ID when multiple sessions are active",
				},
			},
		},
		async execute(_toolCallId, params) {
			const session = (params as { session?: string }).session;
			const result = await runCircuit(session ? ["stop", session] : ["stop"]);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok },
			};
		},
	});

	// -- Slash commands: human affordances --

	pi.registerCommand("circuit", {
		description: "Manage Circuit sessions: /circuit <list|load|scaffold|start|status|advance|stop>",
		handler: async (args, ctx) => {
			const parsed = parseCircuitCommand(args);
			switch (parsed.verb) {
				case "list": {
					const machines = await listMachines();
					ctx.ui.notify(machines.length > 0 ? machines.join("\n") : "No machines found", "info");
					return;
				}
				case "load": {
					if (!parsed.argument) {
						ctx.ui.notify("Usage: /circuit load <machine>", "warning");
						return;
					}
					const result = await runCircuit(["load", parsed.argument]);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
				case "scaffold": {
					if (!parsed.argument) {
						ctx.ui.notify("Usage: /circuit scaffold <machine>", "warning");
						return;
					}
					const result = await runCircuit(["scaffold", parsed.argument]);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
				case "start": {
					if (!parsed.argument) {
						ctx.ui.notify("Usage: /circuit start <machine>", "warning");
						return;
					}
					const result = await runCircuit(["start", parsed.argument]);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
				case "status": {
					const result = await runCircuit(parsed.session ? ["status", parsed.session] : ["status"]);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
				case "advance": {
					if (!parsed.argument) {
						ctx.ui.notify("Usage: /circuit advance <event> [session]", "warning");
						return;
					}
					const result = await runCircuit(
						parsed.session
							? ["advance", parsed.argument, parsed.session]
							: ["advance", parsed.argument],
					);
					ctx.ui.notify(result.message, result.ok ? "info" : "error");
					return;
				}
				case "stop": {
					const result = await runCircuit(parsed.session ? ["stop", parsed.session] : ["stop"]);
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
