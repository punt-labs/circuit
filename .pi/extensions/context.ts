export interface CircuitStatus {
	machine: string;
	current: string;
	enabled: string[];
	blocked: string[];
	checks: Record<string, { result: boolean; invocations: number }>;
}

export function parseCircuitStatus(output: string): CircuitStatus | undefined {
	const lines = output.split("\n");
	const machineLine = lines.find((l) => l.startsWith("machine: "));
	const currentLine = lines.find((l) => l.startsWith("current: "));
	if (!machineLine || !currentLine) return undefined;

	const machine = machineLine.slice("machine: ".length).trim();
	const current = currentLine.slice("current: ".length).trim();
	const enabled: string[] = [];
	const blocked: string[] = [];
	const checks: Record<string, { result: boolean; invocations: number }> = {};

	let section = "";
	for (const line of lines) {
		if (line === "enabled:") {
			section = "enabled";
		} else if (line === "blocked:") {
			section = "blocked";
		} else if (line === "checks:") {
			section = "checks";
		} else if (line.startsWith("  ") && section === "enabled") {
			enabled.push(line.trim());
		} else if (line.startsWith("  ") && section === "blocked") {
			blocked.push(line.trim());
		} else if (line.startsWith("  ") && section === "checks") {
			const match = /^\s+(\w+):\s+(TRUE|FALSE)\s+\(invocations:\s+(\d+)\)/.exec(line);
			if (match) {
				checks[match[1] ?? ""] = {
					result: match[2] === "TRUE",
					invocations: parseInt(match[3] ?? "0", 10),
				};
			}
		}
	}

	return { machine, current, enabled, blocked, checks };
}

export function formatContextInjection(status: CircuitStatus): string {
	const lines: string[] = [
		`Circuit machine: ${status.machine}`,
		`Current state: ${status.current}`,
		"",
	];

	if (status.enabled.length > 0) {
		lines.push("Enabled operations:");
		for (const op of status.enabled) {
			lines.push(`  ${op}`);
		}
	}

	if (status.blocked.length > 0) {
		lines.push("Blocked operations:");
		for (const op of status.blocked) {
			lines.push(`  ${op}`);
		}
	}

	lines.push("");
	lines.push("Use circuit_advance to request a transition. Do not claim");
	lines.push("workflow progress without a successful circuit_advance call.");

	return lines.join("\n");
}

export function formatToolResult(
	allowed: boolean,
	output: string,
): { text: string; isAdvanced: boolean } {
	if (allowed) {
		return { text: output, isAdvanced: true };
	}
	return { text: output, isAdvanced: false };
}

export function parseAdvanceOutput(output: string): {
	allowed: boolean;
	from: string;
	to: string;
	event: string;
} {
	const advancedMatch = /^advanced:\s+(\S+)\s+->\s+(\S+)/.exec(output);
	if (advancedMatch) {
		return {
			allowed: true,
			from: advancedMatch[1] ?? "",
			to: advancedMatch[2] ?? "",
			event: "",
		};
	}
	const blockedMatch = /^blocked:\s+Advance\((\S+)\)/.exec(output);
	if (blockedMatch) {
		return {
			allowed: false,
			from: "",
			to: "",
			event: blockedMatch[1] ?? "",
		};
	}
	return { allowed: false, from: "", to: "", event: "" };
}
