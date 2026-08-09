export interface CircuitStatus {
	session?: string;
	sessionState?: string;
	machine: string;
	current: string;
	enabled: string[];
	blocked: string[];
	checks: Record<string, { result: boolean; invocations: number }>;
}

export function parseCircuitStatus(output: string): CircuitStatus | undefined {
	return parseStatusBlock(output.split("\n"));
}

export function parseCircuitStatuses(output: string): CircuitStatus[] {
	return output
		.split(/\n\s*\n/)
		.map((block) => parseStatusBlock(block.split("\n")))
		.filter((status): status is CircuitStatus => status !== undefined);
}

function parseStatusBlock(lines: string[]): CircuitStatus | undefined {
	const sessionLine = lines.find((line) => line.startsWith("session: "));
	const sessionStateLine = lines.find((line) => line.startsWith("session-state: "));
	const machineLine = lines.find((line) => line.startsWith("machine: "));
	const currentLine = lines.find((line) => line.startsWith("current: "));
	if (!machineLine || !currentLine) return undefined;

	const session = sessionLine?.slice("session: ".length).trim();
	const sessionState = sessionStateLine?.slice("session-state: ".length).trim();
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

	const status = { machine, current, enabled, blocked, checks };
	return {
		...(session ? { session } : {}),
		...(sessionState ? { sessionState } : {}),
		...status,
	};
}

export function formatContextInjection(status: CircuitStatus): string {
	const lines: string[] = [];
	if (status.session) {
		lines.push(`Circuit session: ${status.session}`);
	}
	lines.push(`Circuit machine: ${status.machine}`);
	lines.push(`Current state: ${status.current}`);
	lines.push("");

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

export function formatContextInjections(statuses: CircuitStatus[]): string {
	return statuses.map((status) => formatContextInjection(status)).join("\n\n---\n\n");
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
