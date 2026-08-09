declare module "@earendil-works/pi-coding-agent" {
	export interface ExtensionAPI {
		registerCommand(name: string, command: ExtensionCommand): void;
		registerTool(definition: ToolDefinition): void;
		on(event: "before_agent_start", handler: BeforeAgentStartHandler): void;
		on(event: "session_start", handler: SessionStartHandler): void;
		on(event: string, handler: (...args: unknown[]) => unknown): void;
	}

	export interface ExtensionCommand {
		description: string;
		handler(args: string, ctx: ExtensionCommandContext): Promise<void> | void;
	}

	export interface ExtensionCommandContext {
		ui: ExtensionUI;
	}

	export interface ExtensionUI {
		notify(message: string, level: "info" | "error" | "warning" | "success"): void;
	}

	export interface ToolDefinition {
		name: string;
		label: string;
		description: string;
		promptSnippet?: string;
		promptGuidelines?: string[];
		parameters: unknown;
		execute(
			toolCallId: string,
			params: Record<string, unknown>,
			signal: AbortSignal | undefined,
			onUpdate: ((update: ToolUpdate) => void) | undefined,
			ctx: ToolContext,
		): Promise<ToolResult>;
	}

	export interface ToolUpdate {
		content: ToolContent[];
	}

	export interface ToolResult {
		content: ToolContent[];
		details?: Record<string, unknown>;
	}

	export interface ToolContent {
		type: "text";
		text: string;
	}

	export interface ToolContext {
		cwd: string;
	}

	export type BeforeAgentStartHandler = (
		event: BeforeAgentStartEvent,
		ctx: ExtensionCommandContext,
	) => Promise<BeforeAgentStartResult | undefined> | BeforeAgentStartResult | undefined;

	export interface BeforeAgentStartEvent {
		prompt: string;
		systemPrompt: string;
	}

	export interface BeforeAgentStartResult {
		systemPrompt?: string;
		message?: {
			customType: string;
			content: string;
			display: boolean;
		};
	}

	export type SessionStartHandler = (
		event: SessionStartEvent,
		ctx: ExtensionCommandContext,
	) => Promise<void> | void;

	export interface SessionStartEvent {
		reason: string;
	}
}
