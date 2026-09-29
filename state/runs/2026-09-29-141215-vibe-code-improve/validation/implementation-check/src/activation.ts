import type { CoreHost } from "./core/host";
import { log } from "./log";
import { initializeLanguage, syncLanguageFromSettings } from "./features/language";
import { applyNetworkEnv } from "./features/network-env";
import { createModeStatusBar } from "./features/status-bar";
import { createGoalStatusBar, registerGoalCommands, registerGoalTrackerView } from "./features/goals";
import { registerPlanCommands, registerPlanViews } from "./features/plans";
import { seedDemoScenarios, seedMcpRecommendations, seedSlashCommands, seedVibeignore, seedVibemodes } from "./features/seeds";
import { applyAutonomyPreset, applyDefaultMode, applyTelemetryDefault } from "./features/defaults";
import { initJournal, registerJournalCommands } from "./features/journal";
import { registerAuditNetwork, registerDiagnostics, registerShowContextStats } from "./features/diagnostics";
import { registerTeamConfig } from "./features/team-config";
import { checkForUpdates } from "./features/update-check";
import { showWelcome } from "./features/welcome";
import { registerVibeCodersProxy } from "./features/vibe-coders-proxy";
import { registerOnboarding } from "./features/onboarding";
import { registerPlanCodeLens } from "./features/plan-codelens";
import { registerCommandAudit } from "./features/command-audit";
import { registerUsageReport } from "./features/usage";
import { registerUsageDashboard } from "./features/usage-dashboard";
import { registerGoalLoop } from "./features/goal-loop";
import { registerVerification } from "./features/verification";
import { registerGoalLint } from "./features/goal-lint";
import { checkGoalHealth } from "./features/goal-health";
import { registerGoalCatalog } from "./features/goal-catalog";
import { registerGoalMetrics } from "./features/goal-metrics";
import { registerCheckpoints } from "./features/checkpoints";

type Step = (host: CoreHost) => void | Promise<void>;

/** Commands and views that live under the "aux" umbrella in the original activation log. */
async function registerGoalPlanFeatures(host: CoreHost): Promise<void> {
	registerShowContextStats(host);
	registerJournalCommands(host);
	registerGoalCommands(host);
	registerPlanCommands(host);
	registerGoalTrackerView(host);
	registerPlanViews(host);
	log(host, "aux commands registered (showContextStats, openJournal, openCurrentGoal, showGoalStatus, createGoalHandoff, openLatestPlan)");
}

/**
 * Activation steps in their original order. Each step is isolated: a failure is logged with the
 * step's message and activation continues, so a broken workspace never blocks the core.
 */
const STEPS: Array<[failure: string, run: Step]> = [
	["language sync skipped", syncLanguageFromSettings],
	["proxy/CA setup skipped", applyNetworkEnv],
	["status bar setup failed", createModeStatusBar],
	["goal status bar setup failed", createGoalStatusBar],
	["demo seed skipped", seedDemoScenarios],
	["vibeignore seed skipped", seedVibeignore],
	["telemetry default skipped", applyTelemetryDefault],
	["default mode setup skipped", applyDefaultMode],
	["vibemodes seed skipped", seedVibemodes],
	["autonomy preset skipped", applyAutonomyPreset],
	["journal init skipped", initJournal],
	["aux command registration failed", registerGoalPlanFeatures],
	["slash command seed skipped", seedSlashCommands],
	["mcp seed skipped", seedMcpRecommendations],
	["update check skipped", checkForUpdates],
	["welcome skipped", showWelcome],
	["aux2 command registration failed", registerDiagnostics],
	["team config commands skipped", registerTeamConfig],
	["auditNetwork registration failed", registerAuditNetwork],
	["vibe-coders proxy registration failed", registerVibeCodersProxy],
	["setup wizard registration failed", registerOnboarding],
	["plan CodeLens registration failed", registerPlanCodeLens],
	["command audit registration failed", registerCommandAudit],
	["usage report registration failed", registerUsageReport],
	["usage dashboard registration failed", registerUsageDashboard],
	["goal loop registration failed", registerGoalLoop],
	["verification runner registration failed", registerVerification],
	["goal lint registration failed", registerGoalLint],
	["goal catalog registration failed", registerGoalCatalog],
	["goal metrics registration failed", registerGoalMetrics],
	["checkpoint commands registration failed", registerCheckpoints],
	["goal health check skipped", checkGoalHealth],
	["language default init skipped", initializeLanguage],
];

export async function runActivation(host: CoreHost): Promise<void> {
	for (const [failure, run] of STEPS) {
		try {
			await run(host);
		} catch (error) {
			log(host, `${failure}: ${error}`);
		}
	}
}
