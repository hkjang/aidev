"use strict";

const assert = require("assert");
const fs = require("fs");
const path = require("path");

async function run() {
	const vscode = require("vscode");
	const extensionId = "vibe-code.vibe-code";
	const extension = vscode.extensions.getExtension(extensionId);

	assert.ok(extension, `Expected extension to be discoverable: ${extensionId}`);
	assert.strictEqual(extension.packageJSON.name, "vibe-code");
	assert.strictEqual(extension.packageJSON.publisher, "vibe-code");
	const manifestOnDisk = JSON.parse(fs.readFileSync(path.join(extension.extensionPath, "package.json"), "utf8"));
	assert.strictEqual(extension.packageJSON.version, manifestOnDisk.version);
	assert.match(extension.packageJSON.version, /^\d+\.\d+\.\d+$/);
	assert.ok(fs.existsSync(path.join(extension.extensionPath, "dist", "extension.core.js")), "Expected dist/extension.core.js next to dist/extension.js");
	assert.strictEqual(extension.packageJSON.main, "./dist/extension.js");
	assert.ok(
		Array.isArray(extension.packageJSON.contributes.views["vibe-code-ActivityBar"]),
		"Expected activity bar views to exist"
	);
	assert.ok(
		extension.packageJSON.contributes.views["vibe-code-ActivityBar"].some((view) => view.id === "vibe-code.GoalTracker"),
		"Expected goal tracker view to be contributed"
	);
	assert.ok(
		extension.packageJSON.contributes.views["vibe-code-ActivityBar"].some((view) => view.id === "vibe-code.PlanBoard"),
		"Expected plan board view to be contributed"
	);
	assert.ok(
		extension.packageJSON.contributes.views["vibe-code-ActivityBar"].some((view) => view.id === "vibe-code.GoalPlanMap"),
		"Expected goal plan map view to be contributed"
	);

	const properties = extension.packageJSON.contributes.configuration.properties;
	assert.ok(properties["vibe-code.language"], "Expected vibe-code.language setting");
	assert.strictEqual(properties["vibe-code.language"].default, "ko");
	assert.strictEqual(vscode.workspace.getConfiguration("vibe-code").get("language"), "ko");

	const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
	assert.ok(workspaceRoot, "Expected test workspace to be open");

	const legacyCommandsDir = path.join(workspaceRoot, ".vibe", "commands");
	const legacyGoalPath = path.join(legacyCommandsDir, "목표.md");
	fs.mkdirSync(legacyCommandsDir, { recursive: true });
	fs.writeFileSync(legacyGoalPath, "# legacy /목표\n", "utf8");

	await extension.activate();

	const commands = await vscode.commands.getCommands(true);
	const requiredCommands = [
		"vibe-code.healthCheck",
		"vibe-code.auditNetwork",
		"vibe-code.openJournal",
		"vibe-code.openCurrentGoal",
		"vibe-code.showGoalStatus",
		"vibe-code.createGoalHandoff",
		"vibe-code.openLatestPlan",
		"vibe-code.advanceCurrentPlan",
		"vibe-code.setCurrentPlanStatus",
		"vibe-code.setCurrentPlanPriority",
		"vibe-code.linkCurrentPlanToGoal",
		"vibe-code.archiveDonePlans",
		"vibe-code.restoreArchivedPlan",
		"vibe-code.setActivePlan",
		"vibe-code.selectHighestPriorityPlan",
		"vibe-code.showPlanCatalog",
		"vibe-code.showPlanHistory",
		"vibe-code.showGoalPlanMap",
		"vibe-code.showContextStats",
		"vibe-code.applyVibeCodersProxy",
		"vibe-code.restorePreviousProviderProfile",
		"vibe-code.showVibeCodersProxyStatus",
		"vibe-code.checkVibeCodersProxy",
		"vibe-code.writeVibeCodersProxyConfig",
	];

	for (const command of requiredCommands) {
		assert.ok(commands.includes(command), `Expected command to be registered: ${command}`);
	}

	await vscode.commands.executeCommand("vibe-code.healthCheck");
	await vscode.commands.executeCommand("vibe-code.showVibeCodersProxyStatus");
	await vscode.commands.executeCommand("vibe-code.writeVibeCodersProxyConfig");

	const goalCommandPath = path.join(workspaceRoot, ".vibe", "commands", "goal.md");
	assert.ok(fs.existsSync(goalCommandPath), "Expected /goal slash command to be seeded");
	assert.ok(!fs.existsSync(legacyGoalPath), "Expected legacy /목표 slash command to be migrated away");
	assert.ok(
		fs.readFileSync(goalCommandPath, "utf8").includes(".vibe-code/goals/current.md"),
		"Expected /goal command to describe persistent goal state"
	);

	for (const directoryName of ["goals", "sessions", "checkpoints", "plans", "journal"]) {
		const directoryPath = path.join(workspaceRoot, ".vibe-code", directoryName);
		assert.ok(fs.existsSync(directoryPath), `Expected workspace directory to be created: ${directoryPath}`);
	}

	await vscode.commands.executeCommand("vibe-code.openCurrentGoal");
	await vscode.commands.executeCommand("vibe-code.showGoalStatus");
	await vscode.commands.executeCommand("vibe-code.openLatestPlan");
	await vscode.commands.executeCommand("vibe-code.advanceCurrentPlan");
	await vscode.commands.executeCommand("vibe-code.setCurrentPlanStatus", "blocked");
	await vscode.commands.executeCommand("vibe-code.setCurrentPlanPriority", "P0");
	await vscode.commands.executeCommand("vibe-code.linkCurrentPlanToGoal");
	const currentGoalPath = path.join(workspaceRoot, ".vibe-code", "goals", "current.md");
	const currentPlanPath = path.join(workspaceRoot, ".vibe-code", "plans", "current-plan.md");
	assert.ok(fs.existsSync(currentGoalPath), "Expected current goal file to be created");
	assert.ok(fs.existsSync(currentPlanPath), "Expected current plan file to be created");
	assert.ok(
		fs.readFileSync(currentGoalPath, "utf8").includes("## 작업 큐"),
		"Expected current goal file to include a work queue"
	);
	assert.ok(
		fs.readFileSync(currentPlanPath, "utf8").includes("## Now"),
		"Expected current plan file to include a Now section"
	);
	const currentPlanText = fs.readFileSync(currentPlanPath, "utf8");
	assert.ok(currentPlanText.includes("상태: blocked"), "Expected current plan status to be updated");
	assert.ok(currentPlanText.includes("우선순위: P0"), "Expected current plan priority to be updated");
	assert.ok(currentPlanText.includes("연결 목표: .vibe-code/goals/current.md"), "Expected current plan to link to current goal");
	assert.ok(currentPlanText.includes("연결 목표 제목: 새 목표"), "Expected current plan to include current goal title");
	assert.ok(currentPlanText.includes("## Done"), "Expected current plan to keep a Done section");
	assert.ok(currentPlanText.includes("- [x] 바로 구현할 작업 1개 정의"), "Expected plan advance to move a Now item into Done");
	await vscode.commands.executeCommand("vibe-code.setCurrentPlanStatus", "done");
	await vscode.commands.executeCommand("vibe-code.archiveDonePlans");
	const archivedPlanDir = path.join(workspaceRoot, ".vibe-code", "plans", "archive");
	assert.ok(fs.existsSync(archivedPlanDir), "Expected archived plan directory to be created");
	const archivedPlans = fs.readdirSync(archivedPlanDir).filter((fileName) => fileName.endsWith(".md"));
	assert.ok(archivedPlans.includes("current-plan.md"), "Expected done plan to be archived");
	assert.ok(!fs.existsSync(currentPlanPath), "Expected archived plan to be removed from active plans directory");
	await vscode.commands.executeCommand("vibe-code.restoreArchivedPlan", "current-plan.md");
	assert.ok(fs.existsSync(currentPlanPath), "Expected archived plan to be restored into active plans directory");
	const secondPlanPath = path.join(workspaceRoot, ".vibe-code", "plans", "z-second-plan.md");
	fs.writeFileSync(
		secondPlanPath,
		[
			"# 계획: 두 번째 계획",
			"",
			"상태: draft",
			"우선순위: P1",
			"작성일: 2026-01-01 00:00:00",
			"마지막 갱신: 2026-01-01 00:00:00",
			"",
			"## Now",
			"- [ ] 다른 계획 작업",
			"",
			"## Next",
			"- [ ] 없음",
			"",
			"## Done",
			"",
			"## Risks",
			"- 없음",
			"",
		].join("\n"),
		"utf8"
	);
	await vscode.commands.executeCommand("vibe-code.setActivePlan", "current-plan.md");
	await vscode.commands.executeCommand("vibe-code.setCurrentPlanPriority", "P3");
	await vscode.commands.executeCommand("vibe-code.selectHighestPriorityPlan");
	await vscode.commands.executeCommand("vibe-code.setCurrentPlanStatus", "active");
	await vscode.commands.executeCommand("vibe-code.showPlanCatalog");
	await vscode.commands.executeCommand("vibe-code.showPlanHistory");
	await vscode.commands.executeCommand("vibe-code.showGoalPlanMap");
	assert.ok(
		fs.readFileSync(path.join(workspaceRoot, ".vibe-code", "plans", ".active-plan"), "utf8").trim() ===
			"z-second-plan.md",
		"Expected selectHighestPriorityPlan to select the highest priority active plan"
	);
	assert.ok(
		fs.readFileSync(secondPlanPath, "utf8").includes("상태: active"),
		"Expected highest priority plan to receive active status update"
	);
	assert.ok(
		fs.readFileSync(currentPlanPath, "utf8").includes("우선순위: P3"),
		"Expected lower priority inactive plan to remain available"
	);

	await vscode.commands.executeCommand("vibe-code.createGoalHandoff");
	await vscode.commands.executeCommand("vibe-code.showGoalStatus");
	const handoffFiles = fs
		.readdirSync(path.join(workspaceRoot, ".vibe-code", "sessions"))
		.filter((fileName) => fileName.endsWith("-handoff.md"));
	assert.ok(handoffFiles.length > 0, "Expected a goal handoff file to be created");

	// CodeLens-backed line commands operate on an explicit file + line.
	const lensPlanPath = path.join(workspaceRoot, ".vibe-code", "plans", "lens-plan.md");
	fs.writeFileSync(
		lensPlanPath,
		["# 계획: 렌즈", "", "상태: draft", "작성일: 2026-01-01 00:00:00", "", "## 단계", "- [ ] 단계 A", "", "## Now", "- [ ] 지금 할 일", "", "## Next", "- [ ] 다음 할 일", "", "## Done", ""].join("\n"),
		"utf8"
	);
	const lensUri = vscode.Uri.file(lensPlanPath).toString();
	await vscode.commands.executeCommand("vibe-code.completePlanItem", lensUri, 9);
	let lensText = fs.readFileSync(lensPlanPath, "utf8");
	assert.ok(lensText.includes("## Done\n- [x] 지금 할 일"), "Expected completePlanItem to move the Now line into Done as checked");
	assert.ok(!lensText.includes("- [ ] 지금 할 일"), "Expected completed line to leave Now");
	const nextLine = lensText.split("\n").indexOf("- [ ] 다음 할 일");
	await vscode.commands.executeCommand("vibe-code.promotePlanItem", lensUri, nextLine);
	lensText = fs.readFileSync(lensPlanPath, "utf8");
	assert.ok(/## Now\n- \[ \] 다음 할 일/.test(lensText), "Expected promotePlanItem to move the Next line into Now");
	const stepLine = lensText.split("\n").indexOf("- [ ] 단계 A");
	await vscode.commands.executeCommand("vibe-code.togglePlanCheckbox", lensUri, stepLine);
	lensText = fs.readFileSync(lensPlanPath, "utf8");
	assert.ok(lensText.includes("- [x] 단계 A"), "Expected togglePlanCheckbox to check the step line");
	await vscode.commands.executeCommand("vibe-code.selectPlanFile", lensUri);
	assert.strictEqual(fs.readFileSync(path.join(workspaceRoot, ".vibe-code", "plans", ".active-plan"), "utf8").trim(), "lens-plan.md");
	assert.ok(commands.includes("vibe-code.setupWizard"), "Expected setup wizard command to be registered");
	for (const extra of ["vibe-code.showCommandHistory", "vibe-code.showUsageReport", "vibe-code.openUsageDashboard"]) {
		assert.ok(commands.includes(extra), `Expected command to be registered: ${extra}`);
	}
	// The core reports execute_command lifecycle through globalThis.__vibeCode.onCommand.
	assert.ok(globalThis.__vibeCode && typeof globalThis.__vibeCode.onCommand === "function", "Expected onCommand hook to be installed");
	globalThis.__vibeCode.onCommand({ phase: "approved", command: "echo host-test", cwd: workspaceRoot, taskId: "t1" });
	globalThis.__vibeCode.onCommand({ phase: "exited", command: "echo host-test", cwd: workspaceRoot, exitCode: 0, executionId: "e1", taskId: "t1" });
	await vscode.commands.executeCommand("vibe-code.showCommandHistory");
	// Goal loop hooks: prompt context reflects the current goal; task events are audited.
	assert.strictEqual(typeof globalThis.__vibeCode.promptContext, "function", "Expected promptContext hook");
	const promptText = globalThis.__vibeCode.promptContext();
	assert.ok(promptText.includes("VIBE CODE GOAL STATE") && promptText.includes("새 목표"), "Expected prompt context to describe the current goal");
	globalThis.__vibeCode.onTask({ phase: "started", taskId: "task-1" });
	globalThis.__vibeCode.onTask({ phase: "completed", taskId: "task-1", tokenUsage: { totalTokensIn: 10, totalTokensOut: 5, totalCost: 0.01 }, toolUsage: {}, isSubtask: false });
	assert.ok(commands.includes("vibe-code.resumeGoal"), "Expected resumeGoal command");
	// Verification runner: runs the command on a 검증 계획 line, records 검증 로그 and ticks the line.
	const verifyPlanPath = path.join(workspaceRoot, ".vibe-code", "plans", "verify-plan.md");
	fs.writeFileSync(verifyPlanPath, ["# 계획: 검증", "", "상태: active", "작성일: 2026-01-01 00:00:00", "", "## Now", "- [ ] x", "", "## 검증 계획", "- [ ] `node --version`", ""].join("\n"), "utf8");
	await vscode.commands.executeCommand("vibe-code.runVerification", vscode.Uri.file(verifyPlanPath).toString(), 9);
	const verifyText = fs.readFileSync(verifyPlanPath, "utf8");
	assert.ok(verifyText.includes("- [x] `node --version`"), "Expected passing verification to tick the line");
	assert.ok(/## 검증 로그\n- .+`node --version` → OK/.test(verifyText), "Expected a 검증 로그 entry in the plan");
	assert.ok(/`node --version` → OK/.test(fs.readFileSync(currentGoalPath, "utf8")), "Expected the goal file to receive the verification log too");
	await vscode.commands.executeCommand("vibe-code.setGoalStatus", "active");
	assert.ok(fs.readFileSync(currentGoalPath, "utf8").includes("상태: active"), "Expected setGoalStatus to update the goal status");
	for (const extra of ["vibe-code.runVerificationCommand", "vibe-code.suggestTests", "vibe-code.setGoalStatus", "vibe-code.newGoal", "vibe-code.switchGoal", "vibe-code.linkPlanToCriteria"]) assert.ok(commands.includes(extra), `Expected ${extra}`);
	// Checkpoint before a destructive command (workspace is not a git repo: note only).
	globalThis.__vibeCode.onCommand({ phase: "approved", command: "git reset --hard HEAD~1", cwd: workspaceRoot, taskId: "t2" });
	const checkpointFiles = fs.readdirSync(path.join(workspaceRoot, ".vibe-code", "checkpoints")).filter((n) => n.endsWith(".md"));
	assert.ok(checkpointFiles.length >= 1, "Expected a checkpoint note before a destructive command");
	await vscode.commands.executeCommand("vibe-code.listCheckpoints");
	await vscode.commands.executeCommand("vibe-code.showGoalMetrics");
	await vscode.commands.executeCommand("vibe-code.createRetro");
	const retroFiles = fs.readdirSync(path.join(workspaceRoot, ".vibe-code", "retro")).filter((n) => n.endsWith("-weekly.md"));
	assert.ok(retroFiles.length === 1, "Expected a weekly retro file");
	assert.ok(fs.readFileSync(path.join(workspaceRoot, ".vibe-code", "retro", retroFiles[0]), "utf8").includes("# 주간 회고"), "Expected retro heading");
	// Goal catalog: switching saves the current goal into its own file and loads the other one.
	const goalsDir = path.join(workspaceRoot, ".vibe-code", "goals");
	fs.writeFileSync(path.join(goalsDir, "2026-01-01-0000-second.md"), fs.readFileSync(currentGoalPath, "utf8").replace("# 목표: 새 목표", "# 목표: 두 번째 목표").replace(/^목표 파일:.*$/m, "목표 파일: .vibe-code/goals/2026-01-01-0000-second.md"), "utf8");
	await vscode.commands.executeCommand("vibe-code.switchGoal", "2026-01-01-0000-second.md");
	assert.ok(fs.readFileSync(currentGoalPath, "utf8").includes("# 목표: 두 번째 목표"), "Expected switchGoal to load the chosen goal into current.md");
	const savedPrevious = fs.readdirSync(goalsDir).filter((n) => n !== "current.md" && n !== "2026-01-01-0000-second.md");
	assert.ok(savedPrevious.length >= 1, "Expected the previous current goal to be saved into its own file");
	// Plan -> criteria: a done plan ticks the linked 완료 기준.
	const criteriaPlanPath = path.join(workspaceRoot, ".vibe-code", "plans", "criteria-plan.md");
	fs.writeFileSync(criteriaPlanPath, ["# 계획: 기준", "", "상태: active", "작성일: 2026-01-01 00:00:00", "연결 완료 기준: 1, 3", "", "## 단계", "- [x] a", "", "## Now", "", "## Next", "", "## Done", "- [x] a", ""].join("\n"), "utf8");
	await vscode.commands.executeCommand("vibe-code.setActivePlan", "criteria-plan.md");
	await vscode.commands.executeCommand("vibe-code.setCurrentPlanStatus", "done");
	const criteriaLines = fs.readFileSync(currentGoalPath, "utf8").split("\n").filter((l) => /^- \[( |x)\] /.test(l)).slice(0, 3);
	assert.ok(criteriaLines[0].startsWith("- [x]") && criteriaLines[1].startsWith("- [ ]") && criteriaLines[2].startsWith("- [x]"), `Expected criteria 1 and 3 ticked, got ${criteriaLines.join(" | ")}`);

	const auditDir = path.join(workspaceRoot, ".vibe-code", "audit");
	assert.ok(fs.existsSync(auditDir), "Expected audit directory to be created");
	const auditFiles = fs.readdirSync(auditDir).filter((fileName) => fileName.endsWith(".jsonl"));
	assert.ok(auditFiles.length > 0, "Expected at least one audit JSONL file");
	const auditText = fs.readFileSync(path.join(auditDir, auditFiles[0]), "utf8");
	assert.ok(auditText.includes("\"kind\":\"goal\""), "Expected goal audit entries");
	assert.ok(auditText.includes("\"action\":\"openCurrentGoal\""), "Expected openCurrentGoal audit entry");
	assert.ok(auditText.includes("\"action\":\"showGoalStatus\""), "Expected showGoalStatus audit entry");
	assert.ok(auditText.includes("\"action\":\"createGoalHandoff\""), "Expected createGoalHandoff audit entry");
	assert.ok(auditText.includes("\"action\":\"openLatestPlan\""), "Expected openLatestPlan audit entry");
	assert.ok(auditText.includes("\"action\":\"advanceCurrentPlan\""), "Expected advanceCurrentPlan audit entry");
	assert.ok(auditText.includes("\"action\":\"setCurrentPlanStatus\""), "Expected setCurrentPlanStatus audit entry");
	assert.ok(auditText.includes("\"action\":\"setCurrentPlanPriority\""), "Expected setCurrentPlanPriority audit entry");
	assert.ok(auditText.includes("\"action\":\"linkCurrentPlanToGoal\""), "Expected linkCurrentPlanToGoal audit entry");
	assert.ok(auditText.includes("\"action\":\"archiveDonePlans\""), "Expected archiveDonePlans audit entry");
	assert.ok(auditText.includes("\"action\":\"restoreArchivedPlan\""), "Expected restoreArchivedPlan audit entry");
	assert.ok(auditText.includes("\"action\":\"setActivePlan\""), "Expected setActivePlan audit entry");
	assert.ok(auditText.includes("\"action\":\"selectHighestPriorityPlan\""), "Expected selectHighestPriorityPlan audit entry");
	assert.ok(auditText.includes("\"action\":\"showPlanCatalog\""), "Expected showPlanCatalog audit entry");
	assert.ok(auditText.includes("\"action\":\"showPlanHistory\""), "Expected showPlanHistory audit entry");
	assert.ok(auditText.includes("\"action\":\"showGoalPlanMap\""), "Expected showGoalPlanMap audit entry");
	assert.ok(auditText.includes("\"action\":\"completePlanItem\""), "Expected completePlanItem audit entry");
	assert.ok(auditText.includes("\"kind\":\"command\""), "Expected command audit entries from the onCommand hook");
	assert.ok(auditText.includes("\"action\":\"taskCompleted\""), "Expected taskCompleted audit entry from the onTask hook");
	assert.ok(auditText.includes("\"action\":\"runVerification\""), "Expected runVerification audit entry");
	assert.ok(auditText.includes("\"action\":\"setGoalStatus\""), "Expected setGoalStatus audit entry");
	assert.ok(auditText.includes("\"command\":\"echo host-test\""), "Expected the command text in the audit entry");
	assert.ok(auditText.includes("\"action\":\"promotePlanItem\""), "Expected promotePlanItem audit entry");
	assert.ok(auditText.includes("\"action\":\"togglePlanCheckbox\""), "Expected togglePlanCheckbox audit entry");
	assert.ok(auditText.includes("\"kind\":\"proxy\""), "Expected proxy audit entries");
	assert.ok(auditText.includes("\"action\":\"showVibeCodersProxyStatus\""), "Expected proxy status audit entry");
	assert.ok(auditText.includes("\"action\":\"writeVibeCodersProxyConfig\""), "Expected proxy config audit entry");
}

module.exports = { run };
