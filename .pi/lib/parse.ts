const commands = {
	"machine list": ["", 0, 0],
	"machine check": ["<machine>", 1, 1],
	"machine scaffold": ["<machine>", 1, 1],
	"session start": ["<machine>", 1, 1],
	"session list": ["", 0, 0],
	"session show": ["<session>", 1, 1],
	"session advance": ["<event> [session]", 1, 2],
	"session stop": ["[session]", 0, 1],
	"session remove": ["<session>", 1, 1],
	status: ["", 0, 0],
} as const;

type CircuitCommand = { args: string[] } | { usage: string };

export function parseCircuitCommand(args: string): CircuitCommand {
	const fields = args.trim().split(/\s+/).filter(Boolean);
	if (fields.length === 0) return { args: ["status"] };
	const depth = fields[0] === "machine" || fields[0] === "session" ? 2 : 1;
	const path = fields.slice(0, depth).join(" ");
	if (!Object.hasOwn(commands, path)) {
		return { usage: "Usage: /circuit <machine|session> <command>, or /circuit status" };
	}
	const [operands, minimum, maximum] = commands[path as keyof typeof commands];
	const count = fields.length - depth;
	if (count < minimum || count > maximum) {
		return { usage: `Usage: /circuit ${path} ${operands}`.trimEnd() };
	}
	return { args: fields };
}
