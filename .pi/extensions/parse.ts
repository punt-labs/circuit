export interface CircuitCommand {
	verb: "list" | "start" | "status" | "advance" | "stop";
	argument?: string;
}

export function parseCircuitCommand(args: string): CircuitCommand {
	const fields = args.trim().split(/\s+/).filter(Boolean);
	const verb = fields[0] ?? "status";
	if (verb === "start" || verb === "advance") {
		const argument = fields[1];
		return argument ? { verb, argument } : { verb };
	}
	if (verb === "list" || verb === "status" || verb === "stop") return { verb };
	return { verb: "status" };
}
