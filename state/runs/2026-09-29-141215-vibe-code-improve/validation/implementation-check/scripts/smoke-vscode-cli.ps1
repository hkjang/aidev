param(
	[string]$VsixPath = "",
	[string]$TempRoot = "C:\tmp\vibe-code-vscode-smoke",
	[switch]$Keep = $false
)

$ErrorActionPreference = "Stop"

function Fail([string]$Message) {
	Write-Error $Message
	exit 1
}

function Assert-ChildPath([string]$Parent, [string]$Child) {
	$parentFull = [System.IO.Path]::GetFullPath($Parent).TrimEnd('\', '/')
	$childFull = [System.IO.Path]::GetFullPath($Child)
	if (-not $childFull.StartsWith($parentFull, [System.StringComparison]::OrdinalIgnoreCase)) {
		throw "Refusing to operate outside expected directory: $childFull"
	}
}

function Remove-TreeWithRetry([string]$Path, [int]$Retries = 8, [int]$DelayMs = 500, [switch]$BestEffort) {
	if (-not (Test-Path -LiteralPath $Path)) {
		return $true
	}

	for ($i = 0; $i -lt $Retries; $i++) {
		try {
			Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction Stop
			return $true
		} catch {
			if ($i -lt ($Retries - 1)) {
				Start-Sleep -Milliseconds $DelayMs
			} elseif ($BestEffort.IsPresent) {
				Write-Warning "Could not remove temporary directory yet: $Path"
				Write-Warning $_.Exception.Message
				return $false
			} else {
				throw
			}
		}
	}
}

$root = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$package = Get-Content -LiteralPath (Join-Path $root "package.json") -Raw -Encoding UTF8 | ConvertFrom-Json
$version = [string]$package.version
if (-not $VsixPath) { $VsixPath = "release\vibe-code-$version.vsix" }
$vsixFullPath = [System.IO.Path]::GetFullPath((Join-Path $root $VsixPath))
if (-not (Test-Path -LiteralPath $vsixFullPath)) {
	Fail "VSIX not found: $vsixFullPath"
}

$code = Get-Command code -ErrorAction SilentlyContinue
if (-not $code) {
	Fail "VS Code CLI 'code' was not found in PATH."
}

$tempFull = [System.IO.Path]::GetFullPath($TempRoot)
Assert-ChildPath "C:\tmp" $tempFull
if (Test-Path -LiteralPath $tempFull) {
	Remove-TreeWithRetry $tempFull | Out-Null
}

$extensionsDir = Join-Path $tempFull "extensions"
$userDataDir = Join-Path $tempFull "user-data"
$workspaceDir = Join-Path $tempFull "workspace"
New-Item -ItemType Directory -Path $extensionsDir -Force | Out-Null
New-Item -ItemType Directory -Path $userDataDir -Force | Out-Null
New-Item -ItemType Directory -Path $workspaceDir -Force | Out-Null

Write-Host "Installing VSIX into isolated VS Code directories..."
& code --install-extension $vsixFullPath --extensions-dir $extensionsDir --user-data-dir $userDataDir --force | Out-Host
if ($LASTEXITCODE -ne 0) {
	Fail "VS Code CLI install failed with exit code $LASTEXITCODE"
}

$installed = & code --list-extensions --show-versions --extensions-dir $extensionsDir --user-data-dir $userDataDir
if ($LASTEXITCODE -ne 0) {
	Fail "VS Code CLI extension list failed with exit code $LASTEXITCODE"
}

if (-not ($installed -contains "vibe-code.vibe-code@$version")) {
	Fail "Expected vibe-code.vibe-code@$version in isolated extension list. Got: $($installed -join ', ')"
}

$locateOutput = & code --locate-extension vibe-code.vibe-code --extensions-dir $extensionsDir --user-data-dir $userDataDir 2>&1
$extensionPath = $locateOutput |
	ForEach-Object { [string]$_ } |
	Where-Object { $_ -and (Test-Path -LiteralPath $_) } |
	Select-Object -First 1

if (-not $extensionPath) {
	$extensionPath = Get-ChildItem -LiteralPath $extensionsDir -Directory -ErrorAction SilentlyContinue |
		Where-Object { $_.Name -eq "vibe-code.vibe-code-$version" } |
		Select-Object -ExpandProperty FullName -First 1
}

if (-not $extensionPath) {
	Fail "Could not locate installed vibe-code extension under $extensionsDir"
}

$installedPackagePath = Join-Path $extensionPath "package.json"
if (-not (Test-Path -LiteralPath $installedPackagePath)) {
	Fail "Installed package.json not found: $installedPackagePath"
}

$installedPackage = Get-Content -LiteralPath $installedPackagePath -Raw -Encoding UTF8 | ConvertFrom-Json
$errors = [System.Collections.Generic.List[string]]::new()

if ($installedPackage.name -ne "vibe-code") { $errors.Add("installed package.name must be vibe-code") }
if ($installedPackage.publisher -ne "vibe-code") { $errors.Add("installed package.publisher must be vibe-code") }
if ($installedPackage.version -ne $version) { $errors.Add("installed package.version must be $version") }
if ($installedPackage.main -ne "./dist/extension.js") { $errors.Add("installed package.main must be ./dist/extension.js") }

$languageSetting = $installedPackage.contributes.configuration.properties."vibe-code.language"
if ($languageSetting.default -ne "ko") {
	$errors.Add("installed vibe-code.language default must be ko")
}

$requiredCommands = @(
	"vibe-code.healthCheck",
	"vibe-code.auditNetwork",
	"vibe-code.openJournal",
	"vibe-code.showContextStats"
)

$commands = @($installedPackage.contributes.commands | ForEach-Object { $_.command })
foreach ($command in $requiredCommands) {
	if (-not ($commands -contains $command)) {
		$errors.Add("installed package missing command: $command")
	}
}

$requiredFiles = @(
	"dist\extension.js",
	"dist\extension.core.js",
	"webview-ui\build\assets\index.js",
	"assets\images\vibe-logo.svg",
	"readme.ko.md",
	"docs\source-analysis.md",
	"docs\maintenance-guide.md",
	"docs\improvement-roadmap.md",
	"docs\vibe-coders-proxy.md",
	"scripts\smoke-vscode-cli.ps1",
	"scripts\test-extension-host.ps1",
	"tests\extension-host\index.js"
)

foreach ($relative in $requiredFiles) {
	$path = Join-Path $extensionPath $relative
	if (-not (Test-Path -LiteralPath $path)) {
		$errors.Add("installed package missing file: $relative")
	}
}

$extensionBundle = Join-Path $extensionPath "dist\extension.js"
if (Test-Path -LiteralPath $extensionBundle) {
	$bundleText = Get-Content -LiteralPath $extensionBundle -Raw -Encoding UTF8
	if ($bundleText -notmatch "language default initialized") {
		$errors.Add("installed extension bundle is missing Korean runtime default marker")
	}
}

if ($errors.Count -gt 0) {
	$errors | ForEach-Object { Write-Host "FAIL $_" }
	exit 1
}

if (-not $Keep.IsPresent) {
	Remove-TreeWithRetry $tempFull -BestEffort | Out-Null
}

Write-Host "VS Code CLI smoke test passed: vibe-code.vibe-code@$version"
