package softwarecatalog

import (
	"encoding/json"
	"testing"
)

func observation(category, id, payload string) Observation {
	return Observation{
		Category: category, SourceAssetID: id, Payload: json.RawMessage(payload),
	}
}

func detectionByKey(detections []Detection, key string) (Detection, bool) {
	for _, detection := range detections {
		if detection.ProductKey == key {
			return detection, true
		}
	}
	return Detection{}, false
}

func TestDetectCombinesWindowsProcessServiceAndPackageEvidence(t *testing.T) {
	t.Parallel()
	detections := Detect([]Observation{
		observation("process", "pid-412", `{
			"pid":412,"name":"sqlservr.exe",
			"executable":"C:\\Program Files\\Microsoft SQL Server\\MSSQL16.MSSQLSERVER\\MSSQL\\Binn\\sqlservr.exe"
		}`),
		observation("service", "MSSQLSERVER", `{
			"name":"MSSQLSERVER","display_name":"SQL Server (MSSQLSERVER)",
			"state":"running","active":true,
			"image_path":"\"C:\\Program Files\\Microsoft SQL Server\\MSSQL16.MSSQLSERVER\\MSSQL\\Binn\\sqlservr.exe\" -sMSSQLSERVER"
		}`),
		observation("software.package", "sql-2022", `{
			"name":"Microsoft SQL Server 2022 (64-bit)","version":"16.0.4125.3",
			"publisher":"Microsoft Corporation"
		}`),
	})
	if len(detections) != 1 {
		t.Fatalf("detections = %+v, want exactly SQL Server", detections)
	}
	got := detections[0]
	if got.ProductKey != "microsoft-sql-server" ||
		got.ProductName != "Microsoft SQL Server" ||
		got.Role != "database" || got.Vendor != "Microsoft" {
		t.Fatalf("identity = %+v", got)
	}
	if got.Version != "16.0.4125.3" || got.InstallState != "installed" ||
		got.RuntimeState != "running" || got.ProcessCount != 1 {
		t.Fatalf("state = %+v", got)
	}
	if got.Confidence != 0.99 || got.EvidenceCount != 3 || len(got.Evidence) != 3 {
		t.Fatalf("evidence/confidence = %+v", got)
	}
	if len(got.ExecutablePath) != 1 ||
		got.ExecutablePath[0] != `C:\Program Files\Microsoft SQL Server\MSSQL16.MSSQLSERVER\MSSQL\Binn\sqlservr.exe` {
		t.Fatalf("executable paths = %#v", got.ExecutablePath)
	}
}

func TestDetectRecognizesWindowsIISAndLinuxNginxAliases(t *testing.T) {
	t.Parallel()
	detections := Detect([]Observation{
		observation("service", "w3svc", `{"name":"W3SVC","display_name":"World Wide Web Publishing Service","state":"running"}`),
		observation("process", "nginx-1", `{"name":"nginx","executable":"/usr/sbin/nginx"}`),
	})
	if _, found := detectionByKey(detections, "microsoft-iis"); !found {
		t.Fatalf("W3SVC was not normalized: %+v", detections)
	}
	if nginx, found := detectionByKey(detections, "nginx"); !found ||
		nginx.RuntimeState != "running" || nginx.InstallState != "observed" {
		t.Fatalf("nginx process detection = %+v/%v", nginx, found)
	}
}

func TestDetectIgnoresUnknownAndAmbiguousGenericProcesses(t *testing.T) {
	t.Parallel()
	detections := Detect([]Observation{
		observation("process", "explorer", `{"name":"explorer.exe","executable":"C:\\Windows\\explorer.exe"}`),
		observation("process", "generic-agent", `{"name":"agent.exe","executable":"C:\\Tools\\agent.exe"}`),
		observation("software.package", "pg-client", `{"name":"postgresql-client-16","version":"16.4"}`),
		observation("software.package", "ssms", `{"name":"Microsoft SQL Server Management Studio 20.2","version":"20.2"}`),
		observation("software.package", "sql-native", `{"name":"Microsoft SQL Server 2012 Native Client","version":"11.4"}`),
		observation("software.package", "sql-odbc", `{"name":"Microsoft ODBC Driver 18 for SQL Server","version":"18.5"}`),
		observation("software.package", "sql-setup", `{"name":"Microsoft SQL Server 2022 Setup (English)","version":"16.0"}`),
		observation("software.package", "sql-localdb", `{"name":"Microsoft SQL Server 2022 LocalDB","version":"16.0"}`),
		observation("process", "iis-express", `{"name":"iisexpress.exe","executable":"C:\\Program Files\\IIS Express\\iisexpress.exe"}`),
		observation("software.package", "iis-express-package", `{"name":"IIS Express 10.0","version":"10.0"}`),
	})
	if len(detections) != 0 {
		t.Fatalf("ordinary evidence became products: %+v", detections)
	}
}

func TestPackageOnlyProductUsesUnknownRuntimeState(t *testing.T) {
	t.Parallel()
	detections := Detect([]Observation{
		observation("software.package", "nginx", `{"name":"nginx","version":"1.26.1"}`),
	})
	if len(detections) != 1 {
		t.Fatalf("detections = %+v", detections)
	}
	if detections[0].InstallState != "installed" || detections[0].RuntimeState != "unknown" {
		t.Fatalf("package-only state = %+v", detections[0])
	}
}

func TestRepresentativeVersionUsesNaturalNumericOrdering(t *testing.T) {
	t.Parallel()
	detections := Detect([]Observation{
		observation("software.package", "nginx-old", `{"name":"nginx","version":"9.4"}`),
		observation("software.package", "nginx-new", `{"name":"nginx","version":"16.0"}`),
		observation("software.package", "dotnet-old", `{"name":"dotnet-runtime-8","version":"1.9"}`),
		observation("software.package", "dotnet-new", `{"name":"dotnet-runtime-8","version":"1.10"}`),
	})
	nginx, found := detectionByKey(detections, "nginx")
	if !found || nginx.Version != "16.0" || len(nginx.Versions) != 2 ||
		nginx.Versions[0] != "9.4" || nginx.Versions[1] != "16.0" {
		t.Fatalf("natural nginx versions = %+v/%v", nginx, found)
	}
	dotnet, found := detectionByKey(detections, "dotnet-runtime")
	if !found || dotnet.Version != "1.10" || len(dotnet.Versions) != 2 ||
		dotnet.Versions[0] != "1.9" || dotnet.Versions[1] != "1.10" {
		t.Fatalf("natural .NET versions = %+v/%v", dotnet, found)
	}
}

func TestServiceImagePathDropsArgumentsBeforePersistence(t *testing.T) {
	t.Parallel()
	detections := Detect([]Observation{
		observation("service", "postgres", `{
			"name":"custom-db-service","state":"stopped",
			"image_path":"\"C:\\PostgreSQL\\16\\bin\\postgres.exe\" --password super-secret"
		}`),
	})
	if len(detections) != 1 || detections[0].ProductKey != "postgresql" {
		t.Fatalf("path detection = %+v", detections)
	}
	encoded, _ := json.Marshal(detections[0])
	if string(encoded) == "" || contains(string(encoded), "super-secret") {
		t.Fatalf("service arguments leaked into evidence: %s", encoded)
	}
	if detections[0].RuntimeState != "stopped" {
		t.Fatalf("runtime state = %q", detections[0].RuntimeState)
	}
}

func TestDetectsManagedWorkstationAndEnterpriseProductsFromStrongEvidence(t *testing.T) {
	t.Parallel()
	detections := Detect([]Observation{
		observation("software.package", "office", `{"name":"Microsoft 365 Apps for enterprise - en-us","version":"16.0.19127"}`),
		observation("software.package", "chrome", `{"name":"Google Chrome","version":"140.0.7339.81","publisher":"Google LLC"}`),
		observation("process", "edge", `{"name":"msedge.exe","executable":"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"}`),
		observation("software.package", "firefox", `{"name":"Mozilla Firefox (x64 en-US)","version":"142.0"}`),
		observation("process", "teams", `{"name":"ms-teams.exe","executable":"C:\\Program Files\\WindowsApps\\MSTeams_25180.3000\\ms-teams.exe"}`),
		observation("software.package", "zoom", `{"name":"Zoom Workplace (64-bit)","version":"6.5.10"}`),
		observation("software.package", "java", `{"name":"Microsoft Build of OpenJDK with Hotspot 17","version":"17.0.16"}`),
		observation("software.package", "dotnet", `{"name":"Microsoft .NET Runtime - 8.0.20 (x64)","version":"8.0.20"}`),
		observation("process", "sccm", `{"name":"CcmExec.exe","executable":"C:\\Windows\\CCM\\CcmExec.exe"}`),
		observation("service", "tanium", `{"name":"TaniumClient","display_name":"Tanium Client","state":"running"}`),
		observation("process", "bigfix", `{"name":"BESClient.exe","executable":"C:\\Program Files (x86)\\BigFix Enterprise\\BES Client\\BESClient.exe"}`),
		observation("process", "elastic", `{"name":"elastic-agent.exe","executable":"C:\\Program Files\\Elastic\\Agent\\elastic-agent.exe"}`),
		observation("service", "wazuh", `{"name":"WazuhSvc","display_name":"Wazuh Agent","state":"running"}`),
	})
	want := map[string]string{
		"microsoft-office":  "productivity",
		"google-chrome":     "web_browser",
		"microsoft-edge":    "web_browser",
		"mozilla-firefox":   "web_browser",
		"microsoft-teams":   "collaboration",
		"zoom":              "collaboration",
		"java-runtime":      "application_runtime",
		"dotnet-runtime":    "application_runtime",
		"mecm-client":       "asset_management",
		"tanium-client":     "asset_management",
		"hcl-bigfix-client": "asset_management",
		"elastic-agent":     "observability",
		"wazuh-agent":       "security",
	}
	if len(detections) != len(want) {
		t.Fatalf("workstation detections = %+v, want %d products", detections, len(want))
	}
	for key, role := range want {
		detection, found := detectionByKey(detections, key)
		if !found {
			t.Errorf("product %q was not detected", key)
			continue
		}
		if detection.Role != role || detection.Detection != "builtin_catalog" ||
			detection.Confidence < 0.82 {
			t.Errorf("product %q = %+v", key, detection)
		}
	}
}

func TestChromeRequiresChromeSpecificPackageServiceOrPath(t *testing.T) {
	t.Parallel()
	fromPath := Detect([]Observation{
		observation("process", "chrome", `{"name":"chrome.exe","executable":"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"}`),
	})
	if chrome, found := detectionByKey(fromPath, "google-chrome"); !found ||
		chrome.RuntimeState != "running" || chrome.Confidence != 0.84 {
		t.Fatalf("Chrome installation path detection = %+v/%v", chrome, found)
	}

	ordinary := Detect([]Observation{
		observation("process", "embedded-chrome", `{"name":"chrome.exe","executable":"C:\\Program Files\\Acme Electron App\\chrome.exe"}`),
		observation("process", "embedded-java", `{"name":"java.exe","executable":"C:\\Program Files\\Acme App\\runtime\\bin\\java.exe"}`),
		observation("process", "generic-agent", `{"name":"agent.exe","executable":"C:\\Program Files\\Acme App\\agent.exe"}`),
		observation("process", "electron", `{"name":"electron.exe","executable":"C:\\Program Files\\Acme App\\electron.exe"}`),
		observation("process", "office-updater", `{"name":"OfficeClickToRun.exe","executable":"C:\\Program Files\\Common Files\\Microsoft Shared\\ClickToRun\\OfficeClickToRun.exe"}`),
		observation("software.package", "google-updater", `{"name":"Google Update Helper","version":"1.3"}`),
	})
	if len(ordinary) != 0 {
		t.Fatalf("generic or embedded processes became products: %+v", ordinary)
	}
}

func contains(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
