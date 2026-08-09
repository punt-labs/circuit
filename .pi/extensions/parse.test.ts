import { describe, expect, it } from "vitest";
import { parseCircuitCommand } from "./parse.js";

describe("parseCircuitCommand", () => {
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

	it("parses list", () => {
		expect(parseCircuitCommand("list")).toEqual({ verb: "list" });
	});

	it("parses status", () => {
		expect(parseCircuitCommand("status")).toEqual({ verb: "status" });
	});

	it("parses stop", () => {
		expect(parseCircuitCommand("stop")).toEqual({ verb: "stop" });
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
