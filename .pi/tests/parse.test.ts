import { describe, expect, it } from "vitest";
import { parseCircuitCommand } from "../lib/parse.js";

describe("parseCircuitCommand", () => {
	it.each([
		["machine check review-flow", ["machine", "check", "review-flow"]],
		["machine scaffold review-flow", ["machine", "scaffold", "review-flow"]],
		["session start build-job", ["session", "start", "build-job"]],
		["session advance requestReview", ["session", "advance", "requestReview"]],
		["session advance start build-job-a3f8", ["session", "advance", "start", "build-job-a3f8"]],
		["machine list", ["machine", "list"]],
		["status", ["status"]],
		["session list", ["session", "list"]],
		["session show build-job-a3f8", ["session", "show", "build-job-a3f8"]],
		["session stop", ["session", "stop"]],
		["session stop build-job-a3f8", ["session", "stop", "build-job-a3f8"]],
		["session remove build-job-a3f8", ["session", "remove", "build-job-a3f8"]],
		["", ["status"]],
		["   ", ["status"]],
		["  session   start   build-job  ", ["session", "start", "build-job"]],
	])("parses %j", (input, args) => {
		expect(parseCircuitCommand(input)).toEqual({ args });
	});

	it.each([
		["session start", "session start <machine>"],
		["session advance", "session advance <event> [session]"],
		["status build-job-a3f8", "status"],
		["session list unexpected", "session list"],
	])("shows usage for invalid operands in %j", (input, usage) => {
		expect(parseCircuitCommand(input)).toEqual({ usage: `Usage: /circuit ${usage}` });
	});

	it.each(["bogus", "list", "load build-job", "start build-job", "unload build-job-a3f8"])(
		"rejects unknown command %j without falling back to status",
		(input) => {
			expect(parseCircuitCommand(input)).toHaveProperty("usage");
		},
	);
});
