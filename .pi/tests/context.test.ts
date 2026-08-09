import { describe, expect, it } from "vitest";
import {
	formatContextInjection,
	formatContextInjections,
	formatToolResult,
	parseAdvanceOutput,
	parseCircuitStatus,
	parseCircuitStatuses,
} from "../lib/context.js";

describe("parseCircuitStatus", () => {
	it("parses full status output", () => {
		const output = [
			"session: review-flow-a3f8",
			"machine: review-flow",
			"current: coding",
			"enabled:",
			"  Advance(requestReview)",
			"blocked:",
			"  Advance(approve)",
			"checks:",
			"  makeCheckPassed: TRUE (invocations: 2)",
		].join("\n");

		const status = parseCircuitStatus(output);

		expect(status).toEqual({
			session: "review-flow-a3f8",
			machine: "review-flow",
			current: "coding",
			enabled: ["Advance(requestReview)"],
			blocked: ["Advance(approve)"],
			checks: { makeCheckPassed: { result: true, invocations: 2 } },
		});
	});

	it("parses status without checks", () => {
		const output = [
			"machine: build-job",
			"current: idle",
			"enabled:",
			"  Advance(start)",
			"blocked:",
			"  Advance(finish)",
		].join("\n");

		const status = parseCircuitStatus(output);

		expect(status).toEqual({
			machine: "build-job",
			current: "idle",
			enabled: ["Advance(start)"],
			blocked: ["Advance(finish)"],
			checks: {},
		});
	});

	it("returns undefined for non-status output", () => {
		expect(parseCircuitStatus("started: build-job")).toBeUndefined();
	});

	it("parses multiple status blocks", () => {
		const output = [
			"session: build-job-a3f8",
			"machine: build-job",
			"current: idle",
			"enabled:",
			"  Advance(start)",
			"blocked:",
			"",
			"session: review-flow-b4c9",
			"machine: review-flow",
			"current: coding",
			"enabled:",
			"blocked:",
			"  Advance(requestReview)",
		].join("\n");

		expect(parseCircuitStatuses(output)).toEqual([
			{
				session: "build-job-a3f8",
				machine: "build-job",
				current: "idle",
				enabled: ["Advance(start)"],
				blocked: [],
				checks: {},
			},
			{
				session: "review-flow-b4c9",
				machine: "review-flow",
				current: "coding",
				enabled: [],
				blocked: ["Advance(requestReview)"],
				checks: {},
			},
		]);
	});

	it("handles empty enabled and blocked", () => {
		const output = ["machine: done-machine", "current: done", "enabled:", "blocked:"].join("\n");

		const status = parseCircuitStatus(output);

		expect(status?.enabled).toEqual([]);
		expect(status?.blocked).toEqual([]);
	});
});

describe("formatContextInjection", () => {
	it("includes machine, state, enabled, and blocked operations", () => {
		const text = formatContextInjection({
			session: "build-job-a3f8",
			machine: "build-job",
			current: "idle",
			enabled: ["Advance(start)"],
			blocked: ["Advance(finish)"],
			checks: {},
		});

		expect(text).toContain("Circuit session: build-job-a3f8");
		expect(text).toContain("Circuit machine: build-job");
		expect(text).toContain("Current state: idle");
		expect(text).toContain("Advance(start)");
		expect(text).toContain("Advance(finish)");
		expect(text).toContain("circuit_advance");
	});

	it("omits sections when empty", () => {
		const text = formatContextInjection({
			machine: "done",
			current: "done",
			enabled: [],
			blocked: [],
			checks: {},
		});

		expect(text).not.toContain("Enabled operations:");
		expect(text).not.toContain("Blocked operations:");
	});

	it("formats multiple status injections", () => {
		const text = formatContextInjections([
			{
				session: "build-job-a3f8",
				machine: "build-job",
				current: "idle",
				enabled: [],
				blocked: [],
				checks: {},
			},
			{
				session: "review-flow-b4c9",
				machine: "review-flow",
				current: "coding",
				enabled: [],
				blocked: [],
				checks: {},
			},
		]);

		expect(text).toContain("Circuit session: build-job-a3f8");
		expect(text).toContain("---");
		expect(text).toContain("Circuit session: review-flow-b4c9");
	});
});

describe("formatToolResult", () => {
	it("reports allowed advance", () => {
		const result = formatToolResult(true, "advanced: idle -> running");
		expect(result.isAdvanced).toBe(true);
		expect(result.text).toContain("advanced");
	});

	it("reports blocked advance", () => {
		const result = formatToolResult(false, "blocked: Advance(finish)");
		expect(result.isAdvanced).toBe(false);
		expect(result.text).toContain("blocked");
	});
});

describe("parseAdvanceOutput", () => {
	it("parses allowed advance", () => {
		expect(parseAdvanceOutput("advanced: idle -> running")).toEqual({
			allowed: true,
			from: "idle",
			to: "running",
			event: "",
		});
	});

	it("parses blocked advance", () => {
		expect(parseAdvanceOutput("blocked: Advance(finish)")).toEqual({
			allowed: false,
			from: "",
			to: "",
			event: "finish",
		});
	});

	it("handles unrecognized output", () => {
		expect(parseAdvanceOutput("something else")).toEqual({
			allowed: false,
			from: "",
			to: "",
			event: "",
		});
	});
});
