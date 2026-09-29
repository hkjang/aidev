param(
	[string]$VsixPath = ""
)

$ErrorActionPreference = "Stop"

function Fail([string]$Message) {
	Write-Error $Message
	exit 1
}

$root = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$packagePath = Join-Path $root "package.json"
$package = Get-Content -LiteralPath $packagePath -Raw -Encoding UTF8 | ConvertFrom-Json
$errors = [System.Collections.Generic.List[string]]::new()
if (-not $VsixPath) { $VsixPath = "release\vibe-code-$($package.version).vsix" }

if ($package.name -ne "vibe-code") { $errors.Add("package.name must be vibe-code") }
if ($package.publisher -ne "vibe-code") { $errors.Add("package.publisher must be vibe-code") }
if ($package.version -notmatch "^[0-9]+\.[0-9]+\.[0-9]+$") { $errors.Add("package.version must be semver: $($package.version)") }
$vsixManifestPath = Join-Path $root ".vsixmanifest"
if (Test-Path -LiteralPath $vsixManifestPath) {
	$vsixManifest = Get-Content -LiteralPath $vsixManifestPath -Raw -Encoding UTF8
	if ($vsixManifest -notmatch ("Version=""" + [regex]::Escape($package.version) + """")) { $errors.Add(".vsixmanifest version must match package.json ($($package.version)); re-run scripts/package-vsix.ps1") }
}
if ($package.main -ne "./dist/extension.js") { $errors.Add("package.main must point to ./dist/extension.js") }

$activityBarId = $package.contributes.viewsContainers.activitybar[0].id
if ($activityBarId -ne "vibe-code-ActivityBar") { $errors.Add("activity bar id must be vibe-code-ActivityBar") }
if (-not $package.contributes.views."vibe-code-ActivityBar") { $errors.Add("views must be keyed by vibe-code-ActivityBar") }
if (-not ($package.contributes.views."vibe-code-ActivityBar" | Where-Object { $_.id -eq "vibe-code.GoalTracker" })) {
	$errors.Add("goal tracker view must be contributed in vibe-code-ActivityBar")
}
if (-not ($package.contributes.views."vibe-code-ActivityBar" | Where-Object { $_.id -eq "vibe-code.PlanBoard" })) {
	$errors.Add("plan board view must be contributed in vibe-code-ActivityBar")
}
if (-not ($package.contributes.views."vibe-code-ActivityBar" | Where-Object { $_.id -eq "vibe-code.GoalPlanMap" })) {
	$errors.Add("goal plan map view must be contributed in vibe-code-ActivityBar")
}

foreach ($command in $package.contributes.commands) {
	if (-not ([string]$command.command).StartsWith("vibe-code.")) {
		$errors.Add("command has wrong prefix: $($command.command)")
	}
}

$settings = $package.contributes.configuration.properties.PSObject.Properties
foreach ($setting in $settings) {
	if (-not ([string]$setting.Name).StartsWith("vibe-code.")) {
		$errors.Add("setting has wrong prefix: $($setting.Name)")
	}
}

$languageSetting = $package.contributes.configuration.properties."vibe-code.language"
if ($languageSetting.default -ne "ko") {
	$errors.Add("vibe-code.language default must be ko")
}

$commandIds = @($package.contributes.commands | ForEach-Object { $_.command })
$requiredCommands = @(
	"vibe-code.applyVibeCodersProxy",
	"vibe-code.restorePreviousProviderProfile",
	"vibe-code.showVibeCodersProxyStatus",
	"vibe-code.checkVibeCodersProxy",
	"vibe-code.writeVibeCodersProxyConfig",
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
	"vibe-code.showGoalPlanMap"
)
foreach ($command in $requiredCommands) {
	if (-not ($commandIds -contains $command)) {
		$errors.Add("missing required command: $command")
	}
}

$requiredSettings = @{
	"vibe-code.vibeCodersBaseUrl" = "http://localhost:8080/v1"
	"vibe-code.vibeCodersDefaultModel" = "gpt-4.1-mini"
	"vibe-code.vibeCodersProfileName" = "vibe-coders proxy"
}
foreach ($settingName in $requiredSettings.Keys) {
	$setting = $package.contributes.configuration.properties.$settingName
	if (-not $setting) {
		$errors.Add("missing required setting: $settingName")
	} elseif ($setting.default -ne $requiredSettings[$settingName]) {
		$errors.Add("$settingName default must be $($requiredSettings[$settingName])")
	}
}

$forbiddenPattern = "[Aa][Tt][Hh][Ee][Nn][Aa]|[Uu]racle"
$forbiddenGatewayPattern = (
	"startVibeCoders" + "Gateway|" +
	"openVibeCoders" + "Admin|" +
	"vibeCoders" + "AutoStart|" +
	"vibeCoders" + "ListenAddr|" +
	"vibeCoders" + "AdminUrl"
)
$scanTargets = @(
	"package.json",
	"package.nls.json",
	"package.nls.ko.json",
	"readme.md",
	"readme.ko.md",
	"changelog.md",
	"docs\README.md",
	"docs\source-analysis.md",
	"docs\maintenance-guide.md",
	"docs\improvement-roadmap.md",
	"docs\vibe-coders-proxy.md",
	"docs\autonomous-goal-workflow.md",
	"scripts\package-vsix.ps1",
	"scripts\verify-package.ps1",
	"scripts\smoke-vscode-cli.ps1",
	"scripts\test-extension-host.ps1",
	"scripts\run-extension-host-test.js",
	"tests\extension-host\index.js",
	"dist\extension.js",
	"dist\extension.core.js",
	"webview-ui\build\assets\index.js"
)

foreach ($relative in $scanTargets) {
	$path = Join-Path $root $relative
	if (Test-Path -LiteralPath $path) {
		$text = Get-Content -LiteralPath $path -Raw -Encoding UTF8
		if ($text -match $forbiddenPattern) {
			$errors.Add("stale branding found in $relative")
		}
		if ($text -match $forbiddenGatewayPattern) {
			$errors.Add("gateway-start integration should not be present in $relative")
		}
	}
}

$goalCommandFileName = "goal.md"
$goalCommandPath = Join-Path $root (Join-Path "assets\demo\.vibe\commands" $goalCommandFileName)
if (-not (Test-Path -LiteralPath $goalCommandPath)) {
	$errors.Add("missing Korean /goal slash command template")
} else {
	$goalCommandText = Get-Content -LiteralPath $goalCommandPath -Raw -Encoding UTF8
	if ($goalCommandText -notmatch "\.vibe-code/goals/current\.md") {
		$errors.Add("Korean /goal slash command must describe current goal state file")
	}
	if ($goalCommandText -notmatch "\.vibe-code/sessions/") {
		$errors.Add("Korean /goal slash command must describe session handoff files")
	}
}

$node = Get-Command node -ErrorAction SilentlyContinue
if ($node) {
	& node --check (Join-Path $root "dist\extension.js") | Out-Null
	if ($LASTEXITCODE -ne 0) { $errors.Add("dist/extension.js failed node --check") }
	& node --check (Join-Path $root "dist\extension.core.js") | Out-Null
	if ($LASTEXITCODE -ne 0) { $errors.Add("dist/extension.core.js failed node --check") }
	& node --check (Join-Path $root "webview-ui\build\assets\index.js") | Out-Null
	if ($LASTEXITCODE -ne 0) { $errors.Add("webview-ui/build/assets/index.js failed node --check") }
}

$extensionBundlePath = Join-Path $root "dist\extension.js"
$extensionBundle = Get-Content -LiteralPath $extensionBundlePath -Raw -Encoding UTF8
if ($extensionBundle -notmatch 'require\("\./extension\.core\.js"\)') {
	$errors.Add("dist/extension.js must load ./extension.core.js (run: npm run build)")
}
$coreBundlePath = Join-Path $root "dist\extension.core.js"
if (-not (Test-Path -LiteralPath $coreBundlePath)) {
	$errors.Add("dist/extension.core.js is missing (run: npm run build)")
} else {
	$coreBundle = Get-Content -LiteralPath $coreBundlePath -Raw -Encoding UTF8
	if ($coreBundle -notmatch "__vibeCode\?\.beforeCore") { $errors.Add("core bundle is missing the beforeCore hook (see vendor/PATCHES.md)") }
	if ($coreBundle -notmatch "__vibeCode\?\.mergeLocaleOverrides") { $errors.Add("core bundle is missing the mergeLocaleOverrides hook (see vendor/PATCHES.md)") }
	if (([regex]::Matches($coreBundle, "__vibeCode\?\.onCommand")).Count -ne 3) { $errors.Add("core bundle must call the onCommand hook at 3 sites (see vendor/PATCHES.md)") }
	if (([regex]::Matches($coreBundle, "__vibeCode\?\.onTask")).Count -ne 5) { $errors.Add("core bundle must call the onTask hook at 5 sites (see vendor/PATCHES.md)") }
	if (([regex]::Matches($coreBundle, "__vibeCode\?\.promptContext")).Count -ne 1) { $errors.Add("core bundle must call the promptContext hook once (see vendor/PATCHES.md)") }
	if ($coreBundle -match "\[vibe-code\]") { $errors.Add("core bundle still contains inline feature code; features belong in src/") }
	$vendorCorePath = Join-Path $root "vendor\extension.core.js"
	if ((Test-Path -LiteralPath $vendorCorePath) -and ((Get-Item -LiteralPath $vendorCorePath).Length -ne (Get-Item -LiteralPath $coreBundlePath).Length)) {
		$errors.Add("dist/extension.core.js differs from vendor/extension.core.js (run: npm run build)")
	}
}
if ($extensionBundle -notmatch "goal status bar item created") {
	$errors.Add("active goal status bar integration is missing")
}
if ($extensionBundle -notmatch "vibe-code.openCurrentGoal") {
	$errors.Add("openCurrentGoal runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.showGoalStatus") {
	$errors.Add("showGoalStatus runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.openLatestPlan") {
	$errors.Add("openLatestPlan runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.advanceCurrentPlan") {
	$errors.Add("advanceCurrentPlan runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.setCurrentPlanStatus") {
	$errors.Add("setCurrentPlanStatus runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.setCurrentPlanPriority") {
	$errors.Add("setCurrentPlanPriority runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.linkCurrentPlanToGoal") {
	$errors.Add("linkCurrentPlanToGoal runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.archiveDonePlans") {
	$errors.Add("archiveDonePlans runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.restoreArchivedPlan") {
	$errors.Add("restoreArchivedPlan runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.setActivePlan") {
	$errors.Add("setActivePlan runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.selectHighestPriorityPlan") {
	$errors.Add("selectHighestPriorityPlan runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.showPlanCatalog") {
	$errors.Add("showPlanCatalog runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.showPlanHistory") {
	$errors.Add("showPlanHistory runtime command is missing")
}
if ($extensionBundle -notmatch "vibe-code.showGoalPlanMap") {
	$errors.Add("showGoalPlanMap runtime command is missing")
}
if ($extensionBundle -notmatch "checklist progress") {
	$errors.Add("goal progress reporting is missing")
}
if ($extensionBundle -notmatch "goal tracker view registered") {
	$errors.Add("goal tracker view runtime registration is missing")
}
if ($extensionBundle -notmatch "vibe-code.GoalTracker") {
	$errors.Add("goal tracker view id is missing in runtime bundle")
}
if ($extensionBundle -notmatch "plan board view registered") {
	$errors.Add("plan board view runtime registration is missing")
}
if ($extensionBundle -notmatch "vibe-code.PlanBoard") {
	$errors.Add("plan board view id is missing in runtime bundle")
}
if ($extensionBundle -notmatch "goal plan map view registered") {
	$errors.Add("goal plan map view runtime registration is missing")
}
if ($extensionBundle -notmatch "vibe-code.GoalPlanMap") {
	$errors.Add("goal plan map view id is missing in runtime bundle")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "goal", "showGoalStatus"') {
	$errors.Add("goal audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "openLatestPlan"') {
	$errors.Add("plan audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "advanceCurrentPlan"') {
	$errors.Add("plan advance audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "setCurrentPlanStatus"') {
	$errors.Add("plan status audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "setCurrentPlanPriority"') {
	$errors.Add("plan priority audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "linkCurrentPlanToGoal"') {
	$errors.Add("plan goal-link audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "archiveDonePlans"') {
	$errors.Add("plan archive audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "restoreArchivedPlan"') {
	$errors.Add("plan restore audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "setActivePlan"') {
	$errors.Add("plan active-selection audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "selectHighestPriorityPlan"') {
	$errors.Add("plan highest-priority selection audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "showPlanCatalog"') {
	$errors.Add("plan catalog audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "showPlanHistory"') {
	$errors.Add("plan history audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "plan", "showGoalPlanMap"') {
	$errors.Add("goal-plan map audit logging is missing")
}
if ($extensionBundle -notmatch 'writeAudit\(host, "proxy", "writeVibeCodersProxyConfig"') {
	$errors.Add("proxy audit logging is missing")
}
if ($extensionBundle -notmatch "proxy status bar item created") {
	$errors.Add("proxy status bar integration is missing")
}

$vsixFullPath = [System.IO.Path]::GetFullPath((Join-Path $root $VsixPath))
if (Test-Path -LiteralPath $vsixFullPath) {
	Add-Type -AssemblyName System.IO.Compression.FileSystem
	$zip = [System.IO.Compression.ZipFile]::OpenRead($vsixFullPath)
	try {
		$requiredEntries = @(
			"[Content_Types].xml",
			"extension.vsixmanifest",
			"extension\package.json",
			"extension\dist\extension.js",
			"extension\dist\extension.core.js",
			"extension\docs\source-analysis.md",
			"extension\docs\maintenance-guide.md",
			"extension\docs\improvement-roadmap.md",
			"extension\docs\vibe-coders-proxy.md",
			"extension\docs\autonomous-goal-workflow.md",
			("extension\assets\demo\.vibe\commands\{0}" -f $goalCommandFileName),
			"extension\scripts\package-vsix.ps1",
			"extension\scripts\verify-package.ps1",
			"extension\scripts\smoke-vscode-cli.ps1",
			"extension\scripts\test-extension-host.ps1",
			"extension\scripts\run-extension-host-test.js",
			"extension\tests\extension-host\index.js",
			"extension\webview-ui\build\assets\index.js",
			"extension\assets\images\vibe-logo.svg"
		)
		foreach ($entryName in $requiredEntries) {
			if (-not $zip.GetEntry($entryName)) {
				$errors.Add("VSIX missing $entryName")
			}
		}
	} finally {
		$zip.Dispose()
	}
} else {
	$errors.Add("VSIX not found: $VsixPath")
}

if ($errors.Count -gt 0) {
	$errors | ForEach-Object { Write-Host "FAIL $_" }
	exit 1
}

Write-Host "Vibe Code package verification passed."
