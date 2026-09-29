// Minimal stand-in for the `vscode` module so pure helpers can be unit-tested outside the extension host.
export const version = "test";
export const window = {};
export const workspace = { workspaceFolders: undefined as undefined };
export const commands = {};
export const env = {};
export class TreeItem {
	constructor(
		public label: string,
		public collapsibleState = 0,
	) {}
}
export class ThemeIcon {
	constructor(public id: string) {}
}
export class EventEmitter {
	event = () => ({ dispose() {} });
	fire() {}
	dispose() {}
}
export class RelativePattern {
	constructor(
		public base: unknown,
		public pattern: string,
	) {}
}
export const Uri = { file: (p: string) => ({ fsPath: p }) };
export const StatusBarAlignment = { Left: 1, Right: 2 };
export const TreeItemCollapsibleState = { None: 0, Collapsed: 1, Expanded: 2 };
export const DiagnosticSeverity = { Error: 0, Warning: 1, Information: 2, Hint: 3 };
export class ThemeColor {
	constructor(public id: string) {}
}
export class Range {
	constructor(
		public startLine: number,
		public startCharacter: number,
		public endLine: number,
		public endCharacter: number,
	) {}
}
export class Diagnostic {
	source?: string;
	constructor(
		public range: Range,
		public message: string,
		public severity?: number,
	) {}
}
export const languages = {};
