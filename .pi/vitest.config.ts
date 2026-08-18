import { defineConfig } from "vitest/config";

export default defineConfig({
	test: {
		coverage: {
			provider: "v8",
			reporter: ["text", "json", "json-summary"],
			thresholds: {
				// Ratchet these up via the quality-ratchet tdd-flow pattern.
				statements: 99,
				branches: 86,
				functions: 100,
				lines: 100,
			},
		},
	},
});
