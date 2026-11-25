import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import type { CircuitAdapter, CircuitResult } from "./adapter.js";
import { parseCircuitCommand } from "./parse.js";
import { formatContextInjections, parseAdvanceOutput, parseCircuitStatuses } from "./context.js";

export function registerCircuitExtension(pi: ExtensionAPI, adapter: CircuitAdapter): void {
	pi.on("before_agent_start", async () => {
		const statusResult = await adapter.run(["session", "list"]);
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
		name: "circuit_machine_list",
		label: "Circuit Machine List",
		description: "List available circuit machines.",
		promptSnippet: "List available circuit machines from machines/*.mch",
		parameters: {},
		async execute() {
			const result = await adapter.run(["machine", "list"]);
			if (!result.ok) return textResult(result);
			const machines = result.message ? result.message.split("\n") : [];
			const text = result.message || "No machines found";
			return {
				content: [{ type: "text" as const, text }],
				details: { ok: true, count: machines.length },
			};
		},
	});

	registerMachineTool(pi, adapter, {
		name: "circuit_machine_check",
		label: "Circuit Machine Check",
		description: "Validate a circuit machine and its check bindings without starting a session.",
		promptSnippet: "Validate a circuit machine before starting it",
		path: ["machine", "check"],
	});

	registerMachineTool(pi, adapter, {
		name: "circuit_machine_scaffold",
		label: "Circuit Machine Scaffold",
		description: "Generate missing BOOL check bindings and failing registry stubs for a machine.",
		promptSnippet: "Generate check stubs for a circuit machine",
		path: ["machine", "scaffold"],
	});

	registerMachineTool(pi, adapter, {
		name: "circuit_session_start",
		label: "Circuit Session Start",
		description: "Start an active circuit from a named machine.",
		promptSnippet: "Start a circuit machine to begin a guided workflow",
		path: ["session", "start"],
		promptGuidelines: [
			"Use circuit_machine_list to discover available machines before calling circuit_session_start.",
		],
	});

	pi.registerTool({
		name: "circuit_session_list",
		label: "Circuit Session List",
		description: "List all known circuit sessions and their current states.",
		promptSnippet: "List current circuit sessions and available transitions",
		parameters: {},
		async execute() {
			return textResult(await adapter.run(["session", "list"]));
		},
	});

	pi.registerTool({
		name: "circuit_session_show",
		label: "Circuit Session Show",
		description: "Show the state and enabled and blocked operations for one session.",
		promptSnippet: "Show one circuit session and its available transitions",
		promptGuidelines: [
			"Use circuit_session_show to check the current workflow state before deciding next steps.",
		],
		parameters: {
			type: "object",
			properties: {
				session: {
					type: "string",
					description: "Session ID, e.g. 'build-job-a3f8'",
				},
			},
			required: ["session"],
		},
		async execute(_toolCallId, params) {
			const session = (params as { session?: string }).session;
			if (!session) return missingParam("session");
			return textResult(await adapter.run(["session", "show", session]));
		},
	});

	pi.registerTool({
		name: "circuit_session_advance",
		label: "Circuit Session Advance",
		description:
			"Request a state transition. The B machine validates the precondition and returns allowed or blocked.",
		promptSnippet: "Request a circuit state transition by event name",
		promptGuidelines: [
			"Use circuit_session_advance to request workflow progress. Do not claim a transition succeeded without calling circuit_session_advance first.",
			"Call circuit_session_show before circuit_session_advance if you need to check which operations are enabled.",
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
			const result = await adapter.run(
				session ? ["session", "advance", event, session] : ["session", "advance", event],
			);
			const parsed = parseAdvanceOutput(result.message);
			return {
				content: [{ type: "text" as const, text: result.message }],
				details: { ok: result.ok, allowed: parsed.allowed, from: parsed.from, to: parsed.to },
			};
		},
	});

	pi.registerTool({
		name: "circuit_session_remove",
		label: "Circuit Session Remove",
		description: "Remove a stopped circuit session from runtime storage.",
		promptSnippet: "Remove a stopped circuit session",
		parameters: {
			type: "object",
			properties: {
				session: {
					type: "string",
					description: "Session ID to remove",
				},
			},
			required: ["session"],
		},
		async execute(_toolCallId, params) {
			const session = (params as { session?: string }).session;
			if (!session) return missingParam("session");
			return textResult(await adapter.run(["session", "remove", session]));
		},
	});

	pi.registerTool({
		name: "circuit_session_stop",
		label: "Circuit Session Stop",
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
			return textResult(
				await adapter.run(session ? ["session", "stop", session] : ["session", "stop"]),
			);
		},
	});
}

interface MachineToolConfig {
	name: string;
	label: string;
	description: string;
	promptSnippet: string;
	path: ["machine", "check" | "scaffold"] | ["session", "start"];
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
			return textResult(await adapter.run([...config.path, machine]));
		},
	});
}

function registerSlashCommand(pi: ExtensionAPI, adapter: CircuitAdapter): void {
	pi.registerCommand("circuit", {
		description:
			"Manage Circuit resources: /circuit <machine|session> <command>, or /circuit status",
		handler: async (args, ctx) => {
			const parsed = parseCircuitCommand(args);
			if ("usage" in parsed) {
				ctx.ui.notify(parsed.usage, "warning");
				return;
			}
			await notifyResult(ctx, adapter.run(parsed.args));
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
