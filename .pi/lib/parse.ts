export interface CircuitCommand {
	verb: "list" | "load" | "scaffold" | "start" | "status" | "advance" | "stop";
	argument?: string;
	session?: string;
}

export function parseCircuitCommand(args: string): CircuitCommand {
	const fields = args.trim().split(/\s+/).filter(Boolean);
	const verb = fields[0] ?? "status";
	if (verb === "load" || verb === "scaffold" || verb === "start") {
		const argument = fields[1];
		return argument ? { verb, argument } : { verb };
	}
	if (verb === "advance") {
		const argument = fields[1];
		const session = fields[2];
		if (!argument) return { verb };
		return session ? { verb, argument, session } : { verb, argument };
	}
	if (verb === "status" || verb === "stop") {
		const session = fields[1];
		return session ? { verb, session } : { verb };
	}
	if (verb === "list") return { verb };
	return { verb: "status" };
}
