declare module "@earendil-works/pi-coding-agent" {
	export interface ExtensionAPI {
		registerCommand(name: string, command: ExtensionCommand): void;
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
}
