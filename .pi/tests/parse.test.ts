import { describe, expect, it } from "vitest";
import { parseCircuitCommand } from "../lib/parse.js";

describe("parseCircuitCommand", () => {
	it("parses load with machine name", () => {
		expect(parseCircuitCommand("load review-flow")).toEqual({
			verb: "load",
			argument: "review-flow",
		});
	});

	it("parses scaffold with machine name", () => {
		expect(parseCircuitCommand("scaffold review-flow")).toEqual({
			verb: "scaffold",
			argument: "review-flow",
		});
	});

	it("parses start with machine name", () => {
		expect(parseCircuitCommand("start build-job")).toEqual({
			verb: "start",
			argument: "build-job",
		});
	});

	it("parses start without machine name", () => {
		expect(parseCircuitCommand("start")).toEqual({ verb: "start" });
	});

	it("parses advance with event", () => {
		expect(parseCircuitCommand("advance requestReview")).toEqual({
			verb: "advance",
			argument: "requestReview",
		});
	});

	it("parses advance without event", () => {
		expect(parseCircuitCommand("advance")).toEqual({ verb: "advance" });
	});

	it("parses advance with event and session", () => {
		expect(parseCircuitCommand("advance start build-job-a3f8")).toEqual({
			verb: "advance",
			argument: "start",
			session: "build-job-a3f8",
		});
	});

	it("parses list", () => {
		expect(parseCircuitCommand("list")).toEqual({ verb: "list" });
	});

	it("parses status", () => {
		expect(parseCircuitCommand("status")).toEqual({ verb: "status" });
	});

	it("parses status with session", () => {
		expect(parseCircuitCommand("status build-job-a3f8")).toEqual({
			verb: "status",
			session: "build-job-a3f8",
		});
	});

	it("parses stop", () => {
		expect(parseCircuitCommand("stop")).toEqual({ verb: "stop" });
	});

	it("parses stop with session", () => {
		expect(parseCircuitCommand("stop build-job-a3f8")).toEqual({
			verb: "stop",
			session: "build-job-a3f8",
		});
	});

	it("parses unload with session", () => {
		expect(parseCircuitCommand("unload build-job-a3f8")).toEqual({
			verb: "unload",
			session: "build-job-a3f8",
		});
	});

	it("defaults empty input to status", () => {
		expect(parseCircuitCommand("")).toEqual({ verb: "status" });
	});

	it("defaults whitespace-only input to status", () => {
		expect(parseCircuitCommand("   ")).toEqual({ verb: "status" });
	});

	it("defaults unknown verb to status", () => {
		expect(parseCircuitCommand("bogus")).toEqual({ verb: "status" });
	});

	it("trims leading and trailing whitespace", () => {
		expect(parseCircuitCommand("  start  build-job  ")).toEqual({
			verb: "start",
			argument: "build-job",
		});
	});
});
