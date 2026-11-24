import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

export interface CircuitResult {
	ok: boolean;
	message: string;
}

export interface CircuitAdapter {
	run(args: string[]): Promise<CircuitResult>;
}

export function createCliCircuitAdapter(cwd: string): CircuitAdapter {
	return {
		async run(args: string[]): Promise<CircuitResult> {
			try {
				const result = await execFileAsync("go", ["run", "./cmd/circuit", ...args], { cwd });
				return { ok: true, message: result.stdout.trim() };
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
