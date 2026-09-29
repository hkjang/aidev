import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { isDoneTask, stripTask } from "../util/markdown";
import { workspaceRoot } from "./workspace";

export interface TreeNode extends vscode.TreeItem {
	children?: TreeNode[];
}

export interface ItemOptions {
	description?: string;
	tooltip?: string;
	command?: vscode.Command;
	icon?: string;
	collapsibleState?: vscode.TreeItemCollapsibleState;
	children?: TreeNode[];
}

export function makeItem(label: string, opts: ItemOptions = {}): TreeNode {
	const item: TreeNode = new vscode.TreeItem(label, opts.collapsibleState ?? vscode.TreeItemCollapsibleState.None);
	if (opts.description !== undefined) item.description = opts.description;
	if (opts.tooltip !== undefined) item.tooltip = opts.tooltip;
	if (opts.command) item.command = opts.command;
	if (opts.icon) item.iconPath = new vscode.ThemeIcon(opts.icon);
	if (opts.children) item.children = opts.children;
	return item;
}

export function commandItem(label: string, command: string, title: string, icon: string, args?: unknown[]): TreeNode {
	return makeItem(label, { command: { command, title, arguments: args }, icon });
}

/** Checklist lines as leaf items, or a single placeholder when empty. */
export function checklistItems(lines: string[], emptyLabel: string, icon: string): TreeNode[] {
	if (lines.length === 0) return [makeItem(emptyLabel, { description: "없음", icon: "circle-slash" })];
	return lines.map((line) => {
		const done = isDoneTask(line);
		return makeItem(stripTask(line), { description: done ? "done" : "todo", tooltip: line, icon: done ? "pass" : icon });
	});
}

/** Plain lines as leaf items, or a single placeholder when empty. */
export function noteItems(lines: string[], emptyLabel: string): TreeNode[] {
	if (lines.length === 0) return [makeItem(emptyLabel, { description: "없음", icon: "circle-slash" })];
	return lines.map((line) => makeItem(line, { tooltip: line, icon: "note" }));
}

export interface TreeViewSpec {
	viewId: string;
	roots(): TreeNode[];
	/** Workspace-relative glob patterns whose changes refresh the view. */
	watch: string[];
	refreshMs?: number;
}

/** Register a read-only tree view that rebuilds from `roots()` on a timer and on file changes. */
export function registerTreeView(host: CoreHost, spec: TreeViewSpec): void {
	const emitter = new vscode.EventEmitter<TreeNode | undefined>();
	const provider: vscode.TreeDataProvider<TreeNode> = {
		onDidChangeTreeData: emitter.event,
		getTreeItem: (item) => item,
		getChildren: (item) => {
			const parentId = item?.id ?? spec.viewId;
			const nodes = item ? item.children ?? [] : spec.roots();
			// Stable ids let VS Code keep expand/collapse state across the periodic rebuilds.
			nodes.forEach((node, index) => {
				if (!node.id) node.id = `${parentId}/${index}:${String(node.label)}`;
			});
			return nodes;
		},
	};
	const view = vscode.window.createTreeView(spec.viewId, { treeDataProvider: provider, showCollapseAll: false });
	const refresh = () => emitter.fire(undefined);
	// File watchers are the primary refresh signal; the timer is a fallback for missed events.
	const interval = setInterval(refresh, spec.refreshMs ?? 30_000);
	host.context.subscriptions.push(emitter, view, { dispose: () => clearInterval(interval) });

	const ws = workspaceRoot();
	if (!ws) return;
	for (const pattern of spec.watch) {
		try {
			const watcher = vscode.workspace.createFileSystemWatcher(new vscode.RelativePattern(ws, pattern));
			watcher.onDidChange(refresh);
			watcher.onDidCreate(refresh);
			watcher.onDidDelete(refresh);
			host.context.subscriptions.push(watcher);
		} catch {
			// watcher creation is best-effort; the timer keeps the view fresh
		}
	}
}
