import { describe, expect, it } from "vitest";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import type { CircuitAdapter, CircuitResult } from "../lib/adapter.js";
import { registerCircuitExtension } from "../lib/extension.js";

interface RegisteredTool {
	name: string;
	execute(
		toolCallId: string,
		params: unknown,
	): Promise<{
		content: { type: "text"; text: string }[];
		details: Record<string, unknown>;
	}>;
}

interface RegisteredCommand {
	handler(args: string, ctx: FakeCommandContext): Promise<void>;
}

interface FakeCommandContext {
	ui: { notify(message: string, level: "info" | "error" | "warning"): void };
}

class FakePi {
	readonly tools = new Map<string, RegisteredTool>();
	readonly commands = new Map<string, RegisteredCommand>();
	private beforeAgentStart?: () => Promise<unknown>;

	on(event: string, handler: () => Promise<unknown>): void {
		if (event === "before_agent_start") this.beforeAgentStart = handler;
	}

	registerTool(tool: RegisteredTool): void {
		this.tools.set(tool.name, tool);
	}

	registerCommand(name: string, command: RegisteredCommand): void {
		this.commands.set(name, command);
	}

	async runBeforeAgentStart(): Promise<unknown> {
		return this.beforeAgentStart?.();
	}
}

class FakeAdapter implements CircuitAdapter {
	readonly calls: string[][] = [];
	machines = ["build-job", "review-flow"];
	machineListResult?: CircuitResult;
	result: CircuitResult = { ok: true, message: "ok" };

	run(args: string[]): Promise<CircuitResult> {
		this.calls.push(args);
		if (args.join(" ") === "machine list") {
			return Promise.resolve(
				this.machineListResult ?? { ok: true, message: this.machines.join("\n") },
			);
		}
		return Promise.resolve(this.result);
	}
}

describe("registerCircuitExtension", () => {
	it("registers slash command and all LLM tools", () => {
		const pi = new FakePi();
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());

		expect([...pi.commands.keys()]).toEqual(["circuit"]);
		expect([...pi.tools.keys()].sort()).toEqual([
			"circuit_machine_check",
			"circuit_machine_list",
			"circuit_machine_scaffold",
			"circuit_session_advance",
			"circuit_session_list",
			"circuit_session_remove",
			"circuit_session_show",
			"circuit_session_start",
			"circuit_session_stop",
		]);
	});

	it("injects only active sessions into agent context", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		adapter.result = {
			ok: true,
			message: [
				"session: build-job-a111",
				"session-state: active",
				"machine: build-job",
				"current: running",
				"enabled:",
				"  Advance(finish)",
				"blocked:",
				"",
				"session: build-job-b222",
				"session-state: stopped",
				"machine: build-job",
				"current: done",
				"enabled:",
				"blocked:",
			].join("\n"),
		};
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		const result: unknown = await pi.runBeforeAgentStart();
		expect(adapter.calls).toEqual([["session", "list"]]);

		expect(JSON.stringify(result)).toContain("Circuit session: build-job-a111");
		expect(JSON.stringify(result)).toContain("circuit-state");
		expect(JSON.stringify(result)).not.toContain("build-job-b222");
	});

	it("does not inject when only stopped sessions are known", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		adapter.result = {
			ok: true,
			message: [
				"session: build-job-b222",
				"session-state: stopped",
				"machine: build-job",
				"current: done",
				"enabled:",
				"blocked:",
			].join("\n"),
		};
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		expect(await pi.runBeforeAgentStart()).toBeUndefined();
	});

	it("routes machine tools through adapter", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		await pi.tools.get("circuit_machine_check")?.execute("tool", { machine: "review-flow" });
		await pi.tools.get("circuit_machine_scaffold")?.execute("tool", { machine: "review-flow" });
		await pi.tools.get("circuit_session_start")?.execute("tool", { machine: "review-flow" });

		expect(adapter.calls).toEqual([
			["machine", "check", "review-flow"],
			["machine", "scaffold", "review-flow"],
			["session", "start", "review-flow"],
		]);
	});

	it("reports missing machine parameter for machine tools", async () => {
		const pi = new FakePi();
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());

		const output = await pi.tools.get("circuit_session_start")?.execute("tool", {});

		expect(output?.content[0]?.text).toBe("missing machine parameter");
		expect(output?.details).toEqual({ ok: false });
	});

	it("routes list, status, stop, and advance tools through adapter", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		adapter.result = { ok: true, message: "advanced: idle -> running" };
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		const list = await pi.tools.get("circuit_machine_list")?.execute("tool", {});
		const status = await pi.tools
			.get("circuit_session_show")
			?.execute("tool", { session: "build-job-a111" });
		const stop = await pi.tools
			.get("circuit_session_stop")
			?.execute("tool", { session: "build-job-a111" });
		const unload = await pi.tools.get("circuit_session_remove")?.execute("tool", {
			session: "build-job-a111",
		});
		const advance = await pi.tools.get("circuit_session_advance")?.execute("tool", {
			event: "start",
			session: "build-job-a111",
		});

		expect(list?.content[0]?.text).toBe("build-job\nreview-flow");
		expect(status?.content[0]?.text).toBe("advanced: idle -> running");
		expect(stop?.content[0]?.text).toBe("advanced: idle -> running");
		expect(unload?.content[0]?.text).toBe("advanced: idle -> running");
		expect(advance?.details).toMatchObject({ ok: true, allowed: true });
		expect(adapter.calls).toEqual([
			["machine", "list"],
			["session", "show", "build-job-a111"],
			["session", "stop", "build-job-a111"],
			["session", "remove", "build-job-a111"],
			["session", "advance", "start", "build-job-a111"],
		]);
	});

	it("reports missing event parameter for advance tool", async () => {
		const pi = new FakePi();
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());

		const output = await pi.tools.get("circuit_session_advance")?.execute("tool", {});
		const unload = await pi.tools.get("circuit_session_remove")?.execute("tool", {});
		const show = await pi.tools.get("circuit_session_show")?.execute("tool", {});

		expect(output?.content[0]?.text).toBe("missing event parameter");
		expect(output?.details).toEqual({ ok: false });
		expect(unload?.content[0]?.text).toBe("missing session parameter");
		expect(unload?.details).toEqual({ ok: false });
		expect(show?.content[0]?.text).toBe("missing session parameter");
		expect(show?.details).toEqual({ ok: false });
	});

	it("lists sessions through the session list tool", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		const result = await pi.tools.get("circuit_session_list")?.execute("tool", {});

		expect(result?.content[0]?.text).toBe("ok");
		expect(adapter.calls).toEqual([["session", "list"]]);
	});

	it("distinguishes empty and failed machine listings", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);
		const list = pi.tools.get("circuit_machine_list");

		adapter.machines = [];
		const empty = await list?.execute("tool", {});
		expect(empty?.content[0]?.text).toBe("No machines found");
		expect(empty?.details).toEqual({ ok: true, count: 0 });

		adapter.machineListResult = { ok: false, message: "cannot read machines" };
		const failed = await list?.execute("tool", {});
		expect(failed?.content[0]?.text).toBe("cannot read machines");
		expect(failed?.details).toEqual({ ok: false });
	});

	it("routes tool calls through adapter", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		adapter.result = { ok: true, message: "advanced: idle -> running" };
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		const output = await pi.tools.get("circuit_session_advance")?.execute("tool", {
			event: "start",
			session: "build-job-a111",
		});

		expect(adapter.calls).toEqual([["session", "advance", "start", "build-job-a111"]]);
		expect(output?.content[0]?.text).toBe("advanced: idle -> running");
		expect(output?.details).toMatchObject({ ok: true, allowed: true });
	});

	it("routes slash commands through adapter", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		const notifications: string[] = [];
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);
		const ctx = { ui: { notify: (message: string) => notifications.push(message) } };

		await pi.commands.get("circuit")?.handler("machine list", ctx);
		await pi.commands.get("circuit")?.handler("session list", ctx);
		await pi.commands.get("circuit")?.handler("status", ctx);
		await pi.commands.get("circuit")?.handler("machine check review-flow", ctx);
		await pi.commands.get("circuit")?.handler("machine scaffold review-flow", ctx);
		await pi.commands.get("circuit")?.handler("session start review-flow", ctx);
		await pi.commands.get("circuit")?.handler("session show build-job-a111", ctx);
		await pi.commands.get("circuit")?.handler("session advance start build-job-a111", ctx);
		await pi.commands.get("circuit")?.handler("session stop build-job-a111", ctx);
		await pi.commands.get("circuit")?.handler("session remove build-job-a111", ctx);

		expect(adapter.calls).toEqual([
			["machine", "list"],
			["session", "list"],
			["status"],
			["machine", "check", "review-flow"],
			["machine", "scaffold", "review-flow"],
			["session", "start", "review-flow"],
			["session", "show", "build-job-a111"],
			["session", "advance", "start", "build-job-a111"],
			["session", "stop", "build-job-a111"],
			["session", "remove", "build-job-a111"],
		]);
		expect(notifications).toContain("build-job\nreview-flow");
		expect(notifications).toContain("ok");
	});

	it("warns for incomplete slash commands", async () => {
		const pi = new FakePi();
		const notifications: string[] = [];
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());
		const ctx = { ui: { notify: (message: string) => notifications.push(message) } };

		await pi.commands.get("circuit")?.handler("machine check", ctx);
		await pi.commands.get("circuit")?.handler("machine scaffold", ctx);
		await pi.commands.get("circuit")?.handler("session start", ctx);
		await pi.commands.get("circuit")?.handler("session advance", ctx);
		await pi.commands.get("circuit")?.handler("session remove", ctx);

		expect(notifications).toEqual([
			"Usage: /circuit machine check <machine>",
			"Usage: /circuit machine scaffold <machine>",
			"Usage: /circuit session start <machine>",
			"Usage: /circuit session advance <event> [session]",
			"Usage: /circuit session remove <session>",
		]);
	});
});
