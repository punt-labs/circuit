import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createCliCircuitAdapter } from "../lib/adapter.js";
import { registerCircuitExtension } from "../lib/extension.js";

export default function circuitExtension(pi: ExtensionAPI) {
	registerCircuitExtension(pi, createCliCircuitAdapter(process.cwd()));
}
