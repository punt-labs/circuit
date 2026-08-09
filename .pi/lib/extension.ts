import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import type { CircuitAdapter, CircuitResult } from "./adapter.js";
import { parseCircuitCommand } from "./parse.js";
import { formatContextInjections, parseAdvanceOutput, parseCircuitStatuses } from "./context.js";

export function registerCircuitExtension(pi: ExtensionAPI, adapter: CircuitAdapter): void {
	pi.on("before_agent_start", async () => {
		const statusResult = await adapter.run(["status"]);
		if (!statusResult.ok) return;
		const statuses = parseCircuitStatuses(statusResult.message).filter(isActiveStatus);
		if (statuses.length === 0) return;
		return {
			message: {
				customType: "circuit-state",
				content: formatContextInjections(statuses),
				display: false,
			},
		};
	});

	registerTools(pi, adapter);
	registerSlashCommand(pi, adapter);
}

function isActiveStatus(status: { sessionState?: string }): boolean {
	return status.sessionState === "active" || status.sessionState === undefined;
}

function registerTools(pi: ExtensionAPI, adapter: CircuitAdapter): void {
	pi.registerTool({
		name: "circuit_list",
		label: "Circuit List",
		description: "List available circuit machines.",
		promptSnippet: "List available circuit machines from machines/*.mch",
		parameters: {},
		async execute() {
			const machines = await adapter.listMachines();
			const text = machines.length > 0 ? machines.join("\n") : "No machines found";
			return {
				content: [{ type: "text" as const, text }],
				details: { ok: true, count: machines.length },
			};
		},
	});

	registerMachineTool(pi, adapter, {
		name: "circuit_load",
		label: "Circuit Load",
		description: "Validate a circuit machine and its check bindings without starting a session.",
		promptSnippet: "Validate a circuit machine before starting it",
		verb: "load",
	});

	registerMachineTool(pi, adapter, {
		name: "circuit_scaffold",
		label: "Circuit Scaffold",
		description: "Generate missing BOOL check bindings and failing registry stubs for a machine.",
		promptSnippet: "Generate check stubs for a circuit machine",
		verb: "scaffold",
	});

	registerMachineTool(pi, adapter, {
		name: "circuit_start",
		label: "Circuit Start",
		description: "Start an active circuit from a named machine.",
		promptSnippet: "Start a circuit machine to begin a guided workflow",
		verb: "start",
		promptGuidelines: [
			"Use circuit_list to discover available machines before calling circuit_start.",
		],
	});

	pi.registerTool({
		name: "circuit_status",
		label: "Circuit Status",
		description: "Report known circuit session state, enabled and blocked operations.",
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
			return textResult(await adapter.run(session ? ["status", session] : ["status"]));
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
			if (!event) return missingParam("event");
			const result = await adapter.run(session ? ["advance", event, session] : ["advance", event]);
			const parsed = parseAdvanceOutput(result.message);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok, allowed: parsed.allowed, from: parsed.from, to: parsed.to },
			};
		},
	});

	pi.registerTool({
		name: "circuit_unload",
		label: "Circuit Unload",
		description: "Remove a stopped circuit session from runtime storage.",
		promptSnippet: "Unload a stopped circuit session",
		parameters: {
			type: "object",
			properties: {
				session: {
					type: "string",
					description: "Session ID to unload",
				},
			},
			required: ["session"],
		},
		async execute(_toolCallId, params) {
			const session = (params as { session?: string }).session;
			if (!session) return missingParam("session");
			return textResult(await adapter.run(["unload", session]));
		},
	});

	pi.registerTool({
		name: "circuit_stop",
		label: "Circuit Stop",
		description: "Stop an active circuit session.",
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
			return textResult(await adapter.run(session ? ["stop", session] : ["stop"]));
		},
	});
}

interface MachineToolConfig {
	name: string;
	label: string;
	description: string;
	promptSnippet: string;
	verb: "load" | "scaffold" | "start";
	promptGuidelines?: string[];
}

function registerMachineTool(
	pi: ExtensionAPI,
	adapter: CircuitAdapter,
	config: MachineToolConfig,
): void {
	pi.registerTool({
		name: config.name,
		label: config.label,
		description: config.description,
		promptSnippet: config.promptSnippet,
		...(config.promptGuidelines ? { promptGuidelines: config.promptGuidelines } : {}),
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
			if (!machine) return missingParam("machine");
			return textResult(await adapter.run([config.verb, machine]));
		},
	});
}

function registerSlashCommand(pi: ExtensionAPI, adapter: CircuitAdapter): void {
	pi.registerCommand("circuit", {
		description:
			"Manage Circuit sessions: /circuit <list|load|scaffold|start|status|advance|stop|unload>",
		handler: async (args, ctx) => {
			const parsed = parseCircuitCommand(args);
			switch (parsed.verb) {
				case "list": {
					const machines = await adapter.listMachines();
					ctx.ui.notify(machines.length > 0 ? machines.join("\n") : "No machines found", "info");
					return;
				}
				case "load":
				case "scaffold":
				case "start": {
					if (!parsed.argument) {
						ctx.ui.notify(`Usage: /circuit ${parsed.verb} <machine>`, "warning");
						return;
					}
					await notifyResult(ctx, adapter.run([parsed.verb, parsed.argument]));
					return;
				}
				case "status": {
					await notifyResult(
						ctx,
						adapter.run(parsed.session ? ["status", parsed.session] : ["status"]),
					);
					return;
				}
				case "advance": {
					if (!parsed.argument) {
						ctx.ui.notify("Usage: /circuit advance <event> [session]", "warning");
						return;
					}
					await notifyResult(
						ctx,
						adapter.run(
							parsed.session
								? ["advance", parsed.argument, parsed.session]
								: ["advance", parsed.argument],
						),
					);
					return;
				}
				case "stop": {
					await notifyResult(
						ctx,
						adapter.run(parsed.session ? ["stop", parsed.session] : ["stop"]),
					);
					return;
				}
				case "unload": {
					if (!parsed.session) {
						ctx.ui.notify("Usage: /circuit unload <session>", "warning");
						return;
					}
					await notifyResult(ctx, adapter.run(["unload", parsed.session]));
					return;
				}
			}
		},
	});
}

async function notifyResult(
	ctx: { ui: { notify(message: string, level: "info" | "error" | "warning"): void } },
	resultPromise: Promise<CircuitResult>,
): Promise<void> {
	const result = await resultPromise;
	ctx.ui.notify(result.message, result.ok ? "info" : "error");
}

function textResult(result: CircuitResult): {
	content: { type: "text"; text: string }[];
	details: { ok: boolean };
} {
	return {
		content: [{ type: "text", text: result.message }],
		details: { ok: result.ok },
	};
}

function missingParam(name: string): {
	content: { type: "text"; text: string }[];
	details: { ok: false };
} {
	return {
		content: [{ type: "text", text: `missing ${name} parameter` }],
		details: { ok: false },
	};
}
