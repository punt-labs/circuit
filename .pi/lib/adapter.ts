import { execFile } from "node:child_process";
import { readdir } from "node:fs/promises";
import { basename, join } from "node:path";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const MACHINE_DIR = "machines";

export interface CircuitResult {
	ok: boolean;
	message: string;
}

export interface CircuitAdapter {
	listMachines(): Promise<string[]>;
	run(args: string[]): Promise<CircuitResult>;
}

export function createCliCircuitAdapter(cwd: string): CircuitAdapter {
	return {
		async listMachines(): Promise<string[]> {
			const entries = await readdir(join(cwd, MACHINE_DIR));
			return entries
				.filter((entry) => entry.endsWith(".mch"))
				.map((entry) => basename(entry, ".mch"))
				.sort();
		},
		async run(args: string[]): Promise<CircuitResult> {
			try {
				const result = await execFileAsync("go", ["run", "./cmd/circuit", ...args], { cwd });
				return { ok: true, message: result.stdout.trim() || "circuit command completed" };
			} catch (error) {
				if (isExecError(error)) {
					return {
						ok: false,
						message: error.stderr.trim() || error.stdout.trim() || error.message,
					};
				}
				return { ok: false, message: String(error) };
			}
		},
	};
}

function isExecError(error: unknown): error is Error & { stdout: string; stderr: string } {
	return error instanceof Error && "stdout" in error && "stderr" in error;
}
