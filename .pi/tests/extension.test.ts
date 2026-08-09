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
	result: CircuitResult = { ok: true, message: "ok" };

	listMachines(): Promise<string[]> {
		return Promise.resolve(this.machines);
	}

	run(args: string[]): Promise<CircuitResult> {
		this.calls.push(args);
		return Promise.resolve(this.result);
	}
}

describe("registerCircuitExtension", () => {
	it("registers slash command and all LLM tools", () => {
		const pi = new FakePi();
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());

		expect([...pi.commands.keys()]).toEqual(["circuit"]);
		expect([...pi.tools.keys()].sort()).toEqual([
			"circuit_advance",
			"circuit_list",
			"circuit_load",
			"circuit_scaffold",
			"circuit_start",
			"circuit_status",
			"circuit_stop",
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

		await pi.tools.get("circuit_load")?.execute("tool", { machine: "review-flow" });
		await pi.tools.get("circuit_scaffold")?.execute("tool", { machine: "review-flow" });
		await pi.tools.get("circuit_start")?.execute("tool", { machine: "review-flow" });

		expect(adapter.calls).toEqual([
			["load", "review-flow"],
			["scaffold", "review-flow"],
			["start", "review-flow"],
		]);
	});

	it("reports missing machine parameter for machine tools", async () => {
		const pi = new FakePi();
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());

		const output = await pi.tools.get("circuit_start")?.execute("tool", {});

		expect(output?.content[0]?.text).toBe("missing machine parameter");
		expect(output?.details).toEqual({ ok: false });
	});

	it("routes list, status, stop, and advance tools through adapter", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		adapter.result = { ok: true, message: "advanced: idle -> running" };
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		const list = await pi.tools.get("circuit_list")?.execute("tool", {});
		const status = await pi.tools
			.get("circuit_status")
			?.execute("tool", { session: "build-job-a111" });
		const stop = await pi.tools.get("circuit_stop")?.execute("tool", { session: "build-job-a111" });
		const advance = await pi.tools.get("circuit_advance")?.execute("tool", {
			event: "start",
			session: "build-job-a111",
		});

		expect(list?.content[0]?.text).toBe("build-job\nreview-flow");
		expect(status?.content[0]?.text).toBe("advanced: idle -> running");
		expect(stop?.content[0]?.text).toBe("advanced: idle -> running");
		expect(advance?.details).toMatchObject({ ok: true, allowed: true });
		expect(adapter.calls).toEqual([
			["status", "build-job-a111"],
			["stop", "build-job-a111"],
			["advance", "start", "build-job-a111"],
		]);
	});

	it("reports missing event parameter for advance tool", async () => {
		const pi = new FakePi();
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());

		const output = await pi.tools.get("circuit_advance")?.execute("tool", {});

		expect(output?.content[0]?.text).toBe("missing event parameter");
		expect(output?.details).toEqual({ ok: false });
	});

	it("routes tool calls through adapter", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		adapter.result = { ok: true, message: "advanced: idle -> running" };
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);

		const output = await pi.tools.get("circuit_advance")?.execute("tool", {
			event: "start",
			session: "build-job-a111",
		});

		expect(adapter.calls).toEqual([["advance", "start", "build-job-a111"]]);
		expect(output?.content[0]?.text).toBe("advanced: idle -> running");
		expect(output?.details).toMatchObject({ ok: true, allowed: true });
	});

	it("routes slash commands through adapter", async () => {
		const pi = new FakePi();
		const adapter = new FakeAdapter();
		const notifications: string[] = [];
		registerCircuitExtension(pi as unknown as ExtensionAPI, adapter);
		const ctx = { ui: { notify: (message: string) => notifications.push(message) } };

		await pi.commands.get("circuit")?.handler("list", ctx);
		await pi.commands.get("circuit")?.handler("load review-flow", ctx);
		await pi.commands.get("circuit")?.handler("scaffold review-flow", ctx);
		await pi.commands.get("circuit")?.handler("start review-flow", ctx);
		await pi.commands.get("circuit")?.handler("status build-job-a111", ctx);
		await pi.commands.get("circuit")?.handler("advance start build-job-a111", ctx);
		await pi.commands.get("circuit")?.handler("stop build-job-a111", ctx);

		expect(adapter.calls).toEqual([
			["load", "review-flow"],
			["scaffold", "review-flow"],
			["start", "review-flow"],
			["status", "build-job-a111"],
			["advance", "start", "build-job-a111"],
			["stop", "build-job-a111"],
		]);
		expect(notifications).toContain("build-job\nreview-flow");
		expect(notifications).toContain("ok");
	});

	it("warns for incomplete slash commands", async () => {
		const pi = new FakePi();
		const notifications: string[] = [];
		registerCircuitExtension(pi as unknown as ExtensionAPI, new FakeAdapter());
		const ctx = { ui: { notify: (message: string) => notifications.push(message) } };

		await pi.commands.get("circuit")?.handler("load", ctx);
		await pi.commands.get("circuit")?.handler("scaffold", ctx);
		await pi.commands.get("circuit")?.handler("start", ctx);
		await pi.commands.get("circuit")?.handler("advance", ctx);

		expect(notifications).toEqual([
			"Usage: /circuit load <machine>",
			"Usage: /circuit scaffold <machine>",
			"Usage: /circuit start <machine>",
			"Usage: /circuit advance <event> [session]",
		]);
	});
});
