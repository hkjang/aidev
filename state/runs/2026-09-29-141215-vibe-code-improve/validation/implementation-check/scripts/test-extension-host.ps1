param(
	[string]$TempRoot = (Join-Path "C:\tmp" ("vibe-code-extension-host-test-{0}" -f $PID)),
	[switch]$Keep = $false,
	[switch]$PreferVscodeTestElectron = $false
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
$testEntry = Join-Path $root "tests\extension-host\index.js"
if (-not (Test-Path -LiteralPath $testEntry)) {
	Fail "Extension Host test entry not found: $testEntry"
}

$vscodeTestElectron = Join-Path $root "node_modules\@vscode\test-electron"
if ($PreferVscodeTestElectron.IsPresent -or (Test-Path -LiteralPath $vscodeTestElectron)) {
	Write-Host "Running Extension Host test through @vscode/test-electron..."
	& node (Join-Path $root "scripts\run-extension-host-test.js")
	if ($LASTEXITCODE -ne 0) {
		Fail "@vscode/test-electron run failed with exit code $LASTEXITCODE"
	}
	Write-Host "Extension Host test passed through @vscode/test-electron."
	exit 0
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

Write-Host "@vscode/test-electron not found; using installed VS Code CLI fallback."
Write-Host "Running Extension Host test against development path: $root"

& code `
	--extensionDevelopmentPath $root `
	--extensionTestsPath $testEntry `
	--extensions-dir $extensionsDir `
	--user-data-dir $userDataDir `
	--disable-extensions `
	--disable-gpu `
	--log error `
	$workspaceDir

if ($LASTEXITCODE -ne 0) {
	Fail "Extension Host test failed with exit code $LASTEXITCODE"
}

if (-not $Keep.IsPresent) {
	Remove-TreeWithRetry $tempFull -Retries 20 -DelayMs 750 -BestEffort | Out-Null
}

Write-Host "Extension Host test passed through installed VS Code CLI."
