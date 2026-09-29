param(
	[string]$OutDir = "release",
	[switch]$Lean = $false
)

$ErrorActionPreference = "Stop"
$utf8NoBom = [System.Text.UTF8Encoding]::new($false)

function Escape-Xml([object]$Value) {
	if ($null -eq $Value) {
		return ""
	}

	return [System.Security.SecurityElement]::Escape([string]$Value)
}

function Assert-ChildPath([string]$Parent, [string]$Child) {
	$parentFull = [System.IO.Path]::GetFullPath($Parent).TrimEnd('\', '/')
	$childFull = [System.IO.Path]::GetFullPath($Child)
	if (-not $childFull.StartsWith($parentFull, [System.StringComparison]::OrdinalIgnoreCase)) {
		throw "Refusing to operate outside expected directory: $childFull"
	}
}

$root = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$packagePath = Join-Path $root "package.json"
$package = Get-Content -LiteralPath $packagePath -Raw -Encoding UTF8 | ConvertFrom-Json

$outDirFull = [System.IO.Path]::GetFullPath((Join-Path $root $OutDir))
Assert-ChildPath $root $outDirFull
New-Item -ItemType Directory -Path $outDirFull -Force | Out-Null

$stageRoot = Join-Path "C:\tmp" "vibe-code-vsix-stage"
Assert-ChildPath "C:\tmp" $stageRoot
if (Test-Path -LiteralPath $stageRoot) {
	Remove-Item -LiteralPath $stageRoot -Recurse -Force
}

New-Item -ItemType Directory -Path $stageRoot -Force | Out-Null
$extensionDir = Join-Path $stageRoot "extension"
New-Item -ItemType Directory -Path $extensionDir -Force | Out-Null

$directories = @("assets", "dist", "docs", "integrations", "scripts", "tests", "webview-ui")
foreach ($directory in $directories) {
	Copy-Item -LiteralPath (Join-Path $root $directory) -Destination $extensionDir -Recurse -Force
}

$files = @("package.json", "package.nls.json", "package.nls.ko.json", "readme.md", "readme.ko.md", "changelog.md", "LICENSE.txt")
foreach ($file in $files) {
	Copy-Item -LiteralPath (Join-Path $root $file) -Destination $extensionDir -Force
}

$temporaryFiles = Get-ChildItem -LiteralPath $extensionDir -Recurse -Force -File |
	Where-Object {
		$_.Name -like "*.bak-rewrite" -or
		$_.Name -like "*.tmp" -or
		$_.Name -like "*.tmp-rewrite"
	}
if ($temporaryFiles) {
	foreach ($temporaryFile in $temporaryFiles) {
		[System.IO.File]::Delete($temporaryFile.FullName)
	}
}

# Slim node_modules — remove docs, type defs, source maps, and test directories.
# These are not loaded at runtime and only inflate the VSIX.
$slimNodeModules = Join-Path $extensionDir "dist\node_modules"
if (Test-Path -LiteralPath $slimNodeModules) {
	$slimPatterns = @("*.md", "*.markdown", "*.d.ts", "*.ts", "*.map", "LICENSE", "LICENSE.*", "README", "README.*", "CHANGELOG", "CHANGELOG.*", "HISTORY", "HISTORY.*", ".npmignore", ".eslintrc*", ".prettierrc*", "*.test.js", "*.spec.js")
	foreach ($pattern in $slimPatterns) {
		Get-ChildItem -LiteralPath $slimNodeModules -Recurse -Force -File -Filter $pattern -ErrorAction SilentlyContinue | ForEach-Object {
			# Keep .ts when there is no corresponding .js (rare but possible for some loaders).
			if ($_.Extension -eq ".ts") {
				$jsPath = [System.IO.Path]::ChangeExtension($_.FullName, ".js")
				if (-not (Test-Path -LiteralPath $jsPath)) { return }
			}
			try { [System.IO.File]::Delete($_.FullName) } catch {}
		}
	}
	$testDirs = Get-ChildItem -LiteralPath $slimNodeModules -Recurse -Force -Directory -ErrorAction SilentlyContinue |
		Where-Object { $_.Name -in @("test", "tests", "__tests__", "spec", "specs", "examples", "example", "docs", "doc") }
	foreach ($d in $testDirs) {
		try { Remove-Item -LiteralPath $d.FullName -Recurse -Force -ErrorAction SilentlyContinue } catch {}
	}
}

# Lean mode — drop tree-sitter WASMs for languages rarely used by Korean developers.
# These will simply fail to parse if loaded; tree-sitter handles missing wasm gracefully.
if ($Lean.IsPresent) {
	$leanWasm = @(
		"tree-sitter-tlaplus.wasm",
		"tree-sitter-systemrdl.wasm",
		"tree-sitter-rescript.wasm",
		"tree-sitter-elisp.wasm",
		"tree-sitter-ocaml.wasm",
		"tree-sitter-ql.wasm",
		"tree-sitter-elm.wasm",
		"tree-sitter-solidity.wasm",
		"tree-sitter-scala.wasm",
		"tree-sitter-embedded_template.wasm"
	)
	$distDir = Join-Path $extensionDir "dist"
	foreach ($w in $leanWasm) {
		$p = Join-Path $distDir $w
		if (Test-Path -LiteralPath $p) {
			try { [System.IO.File]::Delete($p); Write-Host "Lean: removed $w" } catch {}
		}
	}
}

$contentTypes = @'
<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="json" ContentType="application/json" />
	<Default Extension="vsixmanifest" ContentType="text/xml" />
	<Default Extension="txt" ContentType="text/plain" />
	<Default Extension="md" ContentType="text/markdown" />
	<Default Extension="png" ContentType="image/png" />
	<Default Extension="svg" ContentType="image/svg+xml" />
	<Default Extension="js" ContentType="application/javascript" />
	<Default Extension="css" ContentType="text/css" />
	<Default Extension="html" ContentType="text/html" />
	<Default Extension="wasm" ContentType="application/wasm" />
	<Default Extension="node" ContentType="application/octet-stream" />
	<Default Extension="ttf" ContentType="application/octet-stream" />
	<Default Extension="woff" ContentType="font/woff" />
	<Default Extension="woff2" ContentType="font/woff2" />
	<Default Extension="map" ContentType="application/json" />
</Types>
'@
[System.IO.File]::WriteAllText((Join-Path $stageRoot "[Content_Types].xml"), $contentTypes, $utf8NoBom)

$tags = ($package.keywords -join ",")
$categories = ($package.categories -join ",")
$homepage = if ($package.homepage) { $package.homepage } else { "" }
$repository = if ($package.repository.url) { $package.repository.url } else { $homepage }

$manifest = @"
<?xml version="1.0" encoding="utf-8"?>
<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011" xmlns:d="http://schemas.microsoft.com/developer/vsx-schema-design/2011">
	<Metadata>
		<Identity Language="en-US" Id="$(Escape-Xml $package.name)" Version="$(Escape-Xml $package.version)" Publisher="$(Escape-Xml $package.publisher)" TargetPlatform="win32-x64" />
		<DisplayName>Vibe Code</DisplayName>
		<Description xml:space="preserve">$(Escape-Xml "AI-powered vibe coding assistant")</Description>
		<Tags>$(Escape-Xml $tags)</Tags>
		<Categories>$(Escape-Xml $categories)</Categories>
		<GalleryFlags>Public</GalleryFlags>
		<Properties>
			<Property Id="Microsoft.VisualStudio.Code.Engine" Value="$(Escape-Xml $package.engines.vscode)" />
			<Property Id="Microsoft.VisualStudio.Code.ExtensionDependencies" Value="" />
			<Property Id="Microsoft.VisualStudio.Code.ExtensionPack" Value="" />
			<Property Id="Microsoft.VisualStudio.Code.ExtensionKind" Value="workspace" />
			<Property Id="Microsoft.VisualStudio.Code.LocalizedLanguages" Value="en,ko" />
			<Property Id="Microsoft.VisualStudio.Code.EnabledApiProposals" Value="" />
			<Property Id="Microsoft.VisualStudio.Code.ExecutesCode" Value="true" />
			<Property Id="Microsoft.VisualStudio.Services.Links.Source" Value="$(Escape-Xml $repository)" />
			<Property Id="Microsoft.VisualStudio.Services.Links.Getstarted" Value="$(Escape-Xml $homepage)" />
			<Property Id="Microsoft.VisualStudio.Services.Links.Repository" Value="$(Escape-Xml $repository)" />
			<Property Id="Microsoft.VisualStudio.Services.Links.Learn" Value="$(Escape-Xml $homepage)" />
			<Property Id="Microsoft.VisualStudio.Services.Branding.Color" Value="$(Escape-Xml $package.galleryBanner.color)" />
			<Property Id="Microsoft.VisualStudio.Services.Branding.Theme" Value="$(Escape-Xml $package.galleryBanner.theme)" />
			<Property Id="Microsoft.VisualStudio.Services.GitHubFlavoredMarkdown" Value="true" />
			<Property Id="Microsoft.VisualStudio.Services.Content.Pricing" Value="Free" />
		</Properties>
		<License>extension/LICENSE.txt</License>
		<Icon>extension/assets/icons/icon.png</Icon>
	</Metadata>
	<Installation>
		<InstallationTarget Id="Microsoft.VisualStudio.Code" />
	</Installation>
	<Dependencies />
	<Assets>
		<Asset Type="Microsoft.VisualStudio.Code.Manifest" Path="extension/package.json" Addressable="true" />
		<Asset Type="Microsoft.VisualStudio.Services.Content.Details" Path="extension/readme.md" Addressable="true" />
		<Asset Type="Microsoft.VisualStudio.Services.Content.Changelog" Path="extension/changelog.md" Addressable="true" />
		<Asset Type="Microsoft.VisualStudio.Services.Content.License" Path="extension/LICENSE.txt" Addressable="true" />
		<Asset Type="Microsoft.VisualStudio.Services.Icons.Default" Path="extension/assets/icons/icon.png" Addressable="true" />
	</Assets>
</PackageManifest>
"@
[System.IO.File]::WriteAllText((Join-Path $stageRoot "extension.vsixmanifest"), $manifest, $utf8NoBom)
[System.IO.File]::WriteAllText((Join-Path $root ".vsixmanifest"), $manifest, $utf8NoBom)

$vsixPath = Join-Path $outDirFull "$($package.name)-$($package.version).vsix"
$zipTemp = "$vsixPath.zip"
if (Test-Path -LiteralPath $vsixPath) {
	Remove-Item -LiteralPath $vsixPath -Force
}
if (Test-Path -LiteralPath $zipTemp) {
	Remove-Item -LiteralPath $zipTemp -Force
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
[System.IO.Compression.ZipFile]::CreateFromDirectory($stageRoot, $zipTemp, [System.IO.Compression.CompressionLevel]::Optimal, $false)
Move-Item -LiteralPath $zipTemp -Destination $vsixPath -Force

Write-Host "Created VSIX: $vsixPath"
