// Package softwarecatalog turns noisy package, service, and process
// observations into a small, host-scoped inventory of recognizable products.
//
// The catalogue is deliberately built into the Server. An operator should not
// have to maintain hundreds of process-name mappings merely to learn that a
// machine runs PostgreSQL or IIS. Detection is deterministic and evidence is
// retained on every result so a false positive can be understood without
// exposing process command lines (which frequently contain credentials).
package softwarecatalog

import (
	"encoding/json"
	"sort"
	"strings"
)

const CatalogVersion = "2026.08.1"

type product struct {
	Key            string
	Name           string
	Vendor         string
	Role           string
	Processes      []string
	Services       []string
	Packages       []string
	Executables    []string
	PathContains   []string
	PackageExclude []string
}

// Catalog focuses on infrastructure and operationally material workstation
// products. It intentionally omits ordinary desktop processes and OS libraries:
// the raw observations remain available, but they do not become pretend CMDB
// services.
var catalog = []product{
	{
		Key: "postgresql", Name: "PostgreSQL", Vendor: "PostgreSQL Global Development Group", Role: "database",
		Processes: []string{"postgres", "postmaster"}, Services: []string{"postgresql*", "postgresql-x64-*", "edb-as-*"},
		Packages: []string{"postgresql*", "edb-as*"}, Executables: []string{"postgres", "postmaster", "pg_ctl"},
		PackageExclude: []string{"client", "common", "libs", "devel", "dev", "contrib", "jdbc", "odbc"},
	},
	{
		Key: "mysql-mariadb", Name: "MySQL / MariaDB", Vendor: "Oracle / MariaDB Foundation", Role: "database",
		Processes: []string{"mysqld", "mariadbd"}, Services: []string{"mysql", "mysqld", "mysql5*", "mysql8*", "mysql9*", "mysql-*", "mariadb", "mariadb*"},
		Packages: []string{"mysql-server*", "mysql-community-server*", "mariadb-server*"}, Executables: []string{"mysqld", "mariadbd"},
	},
	{
		Key: "microsoft-sql-server", Name: "Microsoft SQL Server", Vendor: "Microsoft", Role: "database",
		Processes: []string{"sqlservr"}, Services: []string{"mssqlserver", "mssql$*", "sqlserveragent", "sqlagent$*"},
		Packages: []string{"mssql-server*", "microsoft sql server*"}, Executables: []string{"sqlservr"},
		PackageExclude: []string{"management studio", "native client", "odbc", "jdbc", "browser", "setup support", "setup", "localdb", "compact", "command line utilities", "data tools"},
	},
	{
		Key: "oracle-database", Name: "Oracle Database", Vendor: "Oracle", Role: "database",
		Processes: []string{"oracle", "tnslsnr", "ora_pmon_*", "ora_smon_*"}, Services: []string{"oracleservice*", "oracleoradb*tnslistener*"},
		Packages: []string{"oracle database*", "oracle-database*", "oracle-database-ee*", "oracle-database-xe*"}, Executables: []string{"oracle", "tnslsnr"},
	},
	{
		Key: "mongodb", Name: "MongoDB", Vendor: "MongoDB", Role: "database",
		Processes: []string{"mongod", "mongos"}, Services: []string{"mongodb", "mongod"}, Packages: []string{"mongodb-org-server*", "mongodb-server*"}, Executables: []string{"mongod", "mongos"},
	},
	{
		Key: "redis", Name: "Redis", Vendor: "Redis", Role: "database",
		Processes: []string{"redis-server"}, Services: []string{"redis", "redis-server", "redis@*"}, Packages: []string{"redis-server", "redis"}, Executables: []string{"redis-server"},
	},
	{
		Key: "elasticsearch", Name: "Elasticsearch", Vendor: "Elastic", Role: "search",
		Services: []string{"elasticsearch"}, Packages: []string{"elasticsearch"}, PathContains: []string{"/elasticsearch/", "\\elasticsearch\\"},
	},
	{
		Key: "opensearch", Name: "OpenSearch", Vendor: "OpenSearch Project", Role: "search",
		Services: []string{"opensearch"}, Packages: []string{"opensearch"}, PathContains: []string{"/opensearch/", "\\opensearch\\"},
	},
	{
		Key: "nginx", Name: "NGINX", Vendor: "F5", Role: "web_server",
		Processes: []string{"nginx"}, Services: []string{"nginx"}, Packages: []string{"nginx", "nginx-*", "nginxplus*"}, Executables: []string{"nginx"},
	},
	{
		Key: "apache-http-server", Name: "Apache HTTP Server", Vendor: "Apache Software Foundation", Role: "web_server",
		Processes: []string{"httpd", "apache2"}, Services: []string{"httpd", "apache2"}, Packages: []string{"httpd", "apache2", "apache2-bin"}, Executables: []string{"httpd", "apache2"},
	},
	{
		Key: "microsoft-iis", Name: "Microsoft Internet Information Services", Vendor: "Microsoft", Role: "web_server",
		Processes: []string{"w3wp"}, Services: []string{"w3svc", "was", "iisadmin"}, Executables: []string{"w3wp"},
	},
	{
		Key: "haproxy", Name: "HAProxy", Vendor: "HAProxy Technologies", Role: "reverse_proxy",
		Processes: []string{"haproxy"}, Services: []string{"haproxy"}, Packages: []string{"haproxy"}, Executables: []string{"haproxy"},
	},
	{
		Key: "traefik", Name: "Traefik Proxy", Vendor: "Traefik Labs", Role: "reverse_proxy",
		Processes: []string{"traefik"}, Services: []string{"traefik"}, Packages: []string{"traefik"}, Executables: []string{"traefik"},
	},
	{
		Key: "envoy", Name: "Envoy Proxy", Vendor: "Cloud Native Computing Foundation", Role: "reverse_proxy",
		Processes: []string{"envoy"}, Services: []string{"envoy"}, Packages: []string{"envoy"}, Executables: []string{"envoy"},
	},
	{
		Key: "caddy", Name: "Caddy", Vendor: "Stack Holdings", Role: "web_server",
		Processes: []string{"caddy"}, Services: []string{"caddy"}, Packages: []string{"caddy"}, Executables: []string{"caddy"},
	},
	{
		Key: "apache-tomcat", Name: "Apache Tomcat", Vendor: "Apache Software Foundation", Role: "application_server",
		Services: []string{"tomcat*", "apache tomcat*"}, Packages: []string{"tomcat*", "apache tomcat*"}, PathContains: []string{"/tomcat/", "\\apache tomcat\\"},
	},
	{
		Key: "docker-engine", Name: "Docker Engine", Vendor: "Docker", Role: "container_runtime",
		Processes: []string{"dockerd", "docker desktop"}, Services: []string{"docker", "com.docker.service"},
		Packages: []string{"docker-ce", "docker-ee", "docker.io", "docker desktop*"}, Executables: []string{"dockerd", "com.docker.backend"},
	},
	{
		Key: "containerd", Name: "containerd", Vendor: "Cloud Native Computing Foundation", Role: "container_runtime",
		Processes: []string{"containerd"}, Services: []string{"containerd"}, Packages: []string{"containerd", "containerd.io"}, Executables: []string{"containerd"},
	},
	{
		Key: "podman", Name: "Podman", Vendor: "Red Hat", Role: "container_runtime",
		Processes: []string{"podman"}, Services: []string{"podman", "podman.socket"}, Packages: []string{"podman"}, Executables: []string{"podman"},
	},
	{
		Key: "kubernetes-node", Name: "Kubernetes Node", Vendor: "Cloud Native Computing Foundation", Role: "orchestrator",
		Processes: []string{"kubelet", "kube-proxy"}, Services: []string{"kubelet", "kube-proxy"}, Packages: []string{"kubelet"}, Executables: []string{"kubelet", "kube-proxy"},
	},
	{
		Key: "rabbitmq", Name: "RabbitMQ", Vendor: "Broadcom", Role: "message_broker",
		Processes: []string{"rabbitmq-server"}, Services: []string{"rabbitmq-server", "rabbitmq"}, Packages: []string{"rabbitmq-server*"}, PathContains: []string{"/rabbitmq/", "\\rabbitmq_server-"},
	},
	{
		Key: "apache-kafka", Name: "Apache Kafka", Vendor: "Apache Software Foundation", Role: "message_broker",
		Services: []string{"kafka", "kafka@*", "confluent-kafka*"}, Packages: []string{"confluent-kafka*", "apache kafka*"}, PathContains: []string{"/kafka/", "\\kafka\\"},
	},
	{
		Key: "activemq", Name: "Apache ActiveMQ", Vendor: "Apache Software Foundation", Role: "message_broker",
		Processes: []string{"activemq"}, Services: []string{"activemq", "activemq-artemis"}, Packages: []string{"activemq*"}, Executables: []string{"activemq"},
	},
	{
		Key: "nats-server", Name: "NATS Server", Vendor: "Synadia", Role: "message_broker",
		Processes: []string{"nats-server"}, Services: []string{"nats", "nats-server"}, Packages: []string{"nats-server"}, Executables: []string{"nats-server"},
	},
	{
		Key: "prometheus", Name: "Prometheus", Vendor: "Cloud Native Computing Foundation", Role: "observability",
		Processes: []string{"prometheus"}, Services: []string{"prometheus"}, Packages: []string{"prometheus"}, Executables: []string{"prometheus"},
	},
	{
		Key: "grafana", Name: "Grafana", Vendor: "Grafana Labs", Role: "observability",
		Processes: []string{"grafana", "grafana-server"}, Services: []string{"grafana-server", "grafana"}, Packages: []string{"grafana", "grafana-enterprise"}, Executables: []string{"grafana", "grafana-server"},
	},
	{
		Key: "zabbix-agent", Name: "Zabbix Agent", Vendor: "Zabbix", Role: "observability",
		Processes: []string{"zabbix_agentd", "zabbix_agent2"}, Services: []string{"zabbix-agent", "zabbix-agent2", "zabbix agent*"}, Packages: []string{"zabbix-agent*"}, Executables: []string{"zabbix_agentd", "zabbix_agent2"},
	},
	{
		Key: "splunk", Name: "Splunk", Vendor: "Cisco", Role: "observability",
		Processes: []string{"splunkd"}, Services: []string{"splunkd", "splunkforwarder"}, Packages: []string{"splunk*"}, Executables: []string{"splunkd"},
	},
	{
		Key: "datadog-agent", Name: "Datadog Agent", Vendor: "Datadog", Role: "observability",
		Processes: []string{"datadog-agent"}, Services: []string{"datadog-agent", "datadogagent"}, Packages: []string{"datadog-agent"},
		Executables: []string{"datadog-agent"}, PathContains: []string{"/datadog-agent/", "\\datadog\\datadog agent\\"},
	},
	{
		Key: "crowdstrike-falcon", Name: "CrowdStrike Falcon Sensor", Vendor: "CrowdStrike", Role: "security",
		Processes: []string{"falcon-sensor", "csfalconservice"}, Services: []string{"falcon-sensor", "csfalconservice"}, Packages: []string{"falcon-sensor*", "crowdstrike windows sensor*"}, Executables: []string{"csfalconservice"},
	},
	{
		Key: "sentinelone-agent", Name: "SentinelOne Agent", Vendor: "SentinelOne", Role: "security",
		Processes: []string{"sentinelagent", "sentinelctl"}, Services: []string{"sentinelagent", "sentinel agent*"}, Packages: []string{"sentinelagent*", "sentinelone*"}, Executables: []string{"sentinelagent"},
	},
	{
		Key: "microsoft-defender-endpoint", Name: "Microsoft Defender for Endpoint", Vendor: "Microsoft", Role: "security",
		Processes: []string{"mdatp", "msmpeng", "sense"}, Services: []string{"windefend", "sense"}, Packages: []string{"mdatp"}, Executables: []string{"mdatp", "msmpeng", "senseir"},
	},
	{
		Key: "veeam-agent", Name: "Veeam Agent", Vendor: "Veeam", Role: "backup",
		Processes: []string{"veeamagent", "veeam.endpoint.service"}, Services: []string{"veeam*"}, Packages: []string{"veeam agent*", "veeam-release*"}, Executables: []string{"veeamagent", "veeam.endpoint.service"},
	},
	{
		Key: "jenkins", Name: "Jenkins", Vendor: "Jenkins Project", Role: "ci_cd",
		Services: []string{"jenkins"}, Packages: []string{"jenkins"}, PathContains: []string{"/jenkins/", "\\jenkins\\"},
	},
	{
		Key: "gitlab-runner", Name: "GitLab Runner", Vendor: "GitLab", Role: "ci_cd",
		Processes: []string{"gitlab-runner"}, Services: []string{"gitlab-runner"}, Packages: []string{"gitlab-runner"}, Executables: []string{"gitlab-runner"},
	},
	{
		Key: "openssh-server", Name: "OpenSSH Server", Vendor: "OpenBSD / Microsoft", Role: "remote_access",
		Processes: []string{"sshd"}, Services: []string{"ssh", "sshd"}, Packages: []string{"openssh-server"}, Executables: []string{"sshd"},
	},
	{
		Key: "vmware-tools", Name: "VMware Tools", Vendor: "VMware", Role: "guest_tools",
		Processes: []string{"vmtoolsd"}, Services: []string{"vmtoolsd", "vmtools"}, Packages: []string{"open-vm-tools*", "vmware tools*"}, Executables: []string{"vmtoolsd"},
	},
	{
		Key: "invenqor-agent", Name: "Invenqor Agent", Vendor: "Invenqor", Role: "asset_management",
		Processes: []string{"invenqor-agent"}, Services: []string{"invenqor-agent", "invenqor agent"}, Packages: []string{"invenqor-agent*"}, Executables: []string{"invenqor-agent"},
	},
	{
		// Office applications have many generic process names (WINWORD, EXCEL,
		// OfficeClickToRun). The installed-product record is the authoritative
		// signal and avoids turning an updater process into a suite installation.
		Key: "microsoft-office", Name: "Microsoft 365 / Office", Vendor: "Microsoft", Role: "productivity",
		Packages: []string{"microsoft 365 apps*", "microsoft office professional*", "microsoft office standard*", "microsoft office ltsc*", "microsoft office home and business*"},
	},
	{
		// chrome.exe is also embedded by Electron and Chromium applications. A
		// Chrome-specific package, service or installation path is required.
		Key: "google-chrome", Name: "Google Chrome", Vendor: "Google", Role: "web_browser",
		Services: []string{"googlechromeelevationservice"}, Packages: []string{"google chrome", "google chrome (*", "google-chrome", "google-chrome-stable", "google-chrome-beta"},
		PathContains: []string{"\\google\\chrome\\application\\", "/google/chrome/"},
	},
	{
		Key: "microsoft-edge", Name: "Microsoft Edge", Vendor: "Microsoft", Role: "web_browser",
		Processes: []string{"msedge"}, Services: []string{"microsoftedgeelevationservice", "edgeupdate", "edgeupdatem"},
		Packages:     []string{"microsoft edge", "microsoft edge (*", "microsoft-edge-stable", "microsoft-edge-beta"},
		PathContains: []string{"\\microsoft\\edge\\application\\", "/microsoft/msedge/"},
	},
	{
		Key: "mozilla-firefox", Name: "Mozilla Firefox", Vendor: "Mozilla", Role: "web_browser",
		Processes: []string{"firefox"}, Packages: []string{"mozilla firefox*", "firefox"},
		PathContains: []string{"\\mozilla firefox\\", "/mozilla/firefox/"},
	},
	{
		Key: "microsoft-teams", Name: "Microsoft Teams", Vendor: "Microsoft", Role: "collaboration",
		Processes: []string{"ms-teams", "teams"}, Services: []string{"msteams", "microsoft teams"},
		Packages:     []string{"microsoft teams", "microsoft teams classic*", "teams machine-wide installer*"},
		PathContains: []string{"\\microsoft\\teams\\", "\\windowsapps\\msteams_"},
	},
	{
		Key: "zoom", Name: "Zoom Workplace", Vendor: "Zoom", Role: "collaboration",
		Processes: []string{"zoom"}, Services: []string{"zoomsharingservice", "zoom sharing service"},
		Packages:     []string{"zoom", "zoom workplace*", "zoom (64-bit)*"},
		PathContains: []string{"\\zoom\\bin\\", "/zoom/bin/"},
	},
	{
		// java/javaw alone identifies a runtime but not which distribution and is
		// frequently just an embedded application VM. Require its package or a
		// distribution-specific installation path instead.
		Key: "java-runtime", Name: "Java Runtime / JDK", Vendor: "OpenJDK ecosystem", Role: "application_runtime",
		Packages:     []string{"openjdk-*", "java-*-openjdk*", "temurin-*", "eclipse temurin*", "java * update *", "java se development kit*", "jdk-*", "microsoft build of openjdk*", "oracle jdk*", "amazon corretto*", "ibm semeru*"},
		PathContains: []string{"\\java\\jre", "\\java\\jdk", "\\eclipse adoptium\\", "/jvm/java-", "/java/jre", "/java/jdk"},
	},
	{
		Key: "dotnet-runtime", Name: ".NET Runtime / Hosting Bundle", Vendor: "Microsoft", Role: "application_runtime",
		Processes: []string{"dotnet"}, Packages: []string{"microsoft .net runtime*", "microsoft asp.net core*shared framework*", "microsoft asp.net core*hosting*", ".net*hosting bundle*", "dotnet-runtime-*", "aspnetcore-runtime-*", "dotnet-host", "dotnet-hostfxr-*", "dotnet-sdk-*"},
		PathContains: []string{"\\dotnet\\dotnet.exe", "/usr/share/dotnet/", "/usr/lib/dotnet/"},
	},
	{
		Key: "mecm-client", Name: "Microsoft Configuration Manager Client", Vendor: "Microsoft", Role: "asset_management",
		Processes: []string{"ccmexec"}, Services: []string{"ccmexec", "sms agent host"}, Packages: []string{"configuration manager client*", "microsoft endpoint configuration manager client*"},
		PathContains: []string{"\\windows\\ccm\\ccmexec.exe", "\\microsoft configuration manager\\"},
	},
	{
		Key: "tanium-client", Name: "Tanium Client", Vendor: "Tanium", Role: "asset_management",
		Processes: []string{"taniumclient"}, Services: []string{"taniumclient", "tanium client"}, Packages: []string{"tanium client*"},
		PathContains: []string{"\\tanium\\tanium client\\", "/tanium/taniumclient/"},
	},
	{
		Key: "hcl-bigfix-client", Name: "HCL BigFix Client", Vendor: "HCLSoftware", Role: "asset_management",
		Processes: []string{"besclient"}, Services: []string{"besclient", "bigfix client"}, Packages: []string{"hcl bigfix client*", "ibm bigfix client*", "bigfix client*"},
		PathContains: []string{"\\bigfix enterprise\\bes client\\", "/besclient/", "/var/opt/besclient/"},
	},
	{
		Key: "elastic-agent", Name: "Elastic Agent", Vendor: "Elastic", Role: "observability",
		Processes: []string{"elastic-agent"}, Services: []string{"elastic-agent", "elastic agent"}, Packages: []string{"elastic-agent*"},
		PathContains: []string{"\\elastic\\agent\\", "/elastic-agent/"},
	},
	{
		Key: "wazuh-agent", Name: "Wazuh Agent", Vendor: "Wazuh", Role: "security",
		Processes: []string{"wazuh-agentd", "wazuh-modulesd"}, Services: []string{"wazuh-agent", "wazuhsvc", "wazuh"}, Packages: []string{"wazuh-agent*"},
		PathContains: []string{"\\wazuh-agent\\", "/wazuh-agent/", "/var/ossec/"},
	},
}

type Observation struct {
	Category      string
	SourceAssetID string
	Payload       json.RawMessage
}

type Evidence struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	SourceAssetID string `json:"source_asset_id"`
}

type Detection struct {
	ProductKey     string     `json:"product_key"`
	ProductName    string     `json:"product_name"`
	Role           string     `json:"role"`
	Vendor         string     `json:"vendor"`
	Version        string     `json:"version"`
	Versions       []string   `json:"versions,omitempty"`
	InstallState   string     `json:"install_state"`
	RuntimeState   string     `json:"runtime_state"`
	ServiceNames   []string   `json:"service_names"`
	ProcessNames   []string   `json:"process_names"`
	PackageNames   []string   `json:"package_names"`
	ExecutablePath []string   `json:"executable_paths"`
	Evidence       []Evidence `json:"evidence"`
	EvidenceCount  int        `json:"evidence_count"`
	ProcessCount   int        `json:"process_count"`
	Detection      string     `json:"detection_method"`
	CatalogVersion string     `json:"catalog_version"`
	Confidence     float64    `json:"confidence"`
}

type detectionBuilder struct {
	product        product
	evidence       []Evidence
	seenEvidence   map[string]bool
	kinds          map[string]bool
	versions       map[string]bool
	serviceNames   map[string]bool
	processNames   map[string]bool
	packageNames   map[string]bool
	executablePath map[string]bool
	processCount   int
	running        bool
	serviceState   bool
	serviceStopped bool
	strength       float64
}

const maxEvidence = 32

// Detect maps only known products. Unknown processes are deliberately ignored
// rather than turning every PID into another pseudo-software asset.
func Detect(observations []Observation) []Detection {
	builders := map[string]*detectionBuilder{}
	for _, observation := range observations {
		parsed, ok := parseObservation(observation)
		if !ok {
			continue
		}
		for _, definition := range catalog {
			strength, matched := definition.match(parsed)
			if !matched {
				continue
			}
			builder := builders[definition.Key]
			if builder == nil {
				builder = &detectionBuilder{
					product: definition, seenEvidence: map[string]bool{},
					kinds: map[string]bool{}, versions: map[string]bool{},
					serviceNames: map[string]bool{}, processNames: map[string]bool{},
					packageNames: map[string]bool{}, executablePath: map[string]bool{},
				}
				builders[definition.Key] = builder
			}
			builder.add(parsed, observation.SourceAssetID, strength)
		}
	}
	keys := make([]string, 0, len(builders))
	for key := range builders {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]Detection, 0, len(keys))
	for _, key := range keys {
		result = append(result, builders[key].finish())
	}
	return result
}

type parsedObservation struct {
	kind            string
	name            string
	displayName     string
	version         string
	executablePaths []string
	running         bool
	serviceState    bool
	serviceStopped  bool
}

func parseObservation(observation Observation) (parsedObservation, bool) {
	var payload map[string]any
	if json.Unmarshal(observation.Payload, &payload) != nil {
		return parsedObservation{}, false
	}
	text := func(key string) string {
		value, _ := payload[key].(string)
		return strings.TrimSpace(value)
	}
	parsed := parsedObservation{
		name: text("name"), displayName: text("display_name"),
		version: text("version"),
	}
	switch observation.Category {
	case "process":
		parsed.kind = "process"
		parsed.running = true
		parsed.executablePaths = executablePaths(text("executable"))
	case "service":
		parsed.kind = "service"
		parsed.executablePaths = executablePaths(text("image_path"))
		parsed.running, parsed.serviceState, parsed.serviceStopped = serviceRuntime(payload)
	case "software.package":
		parsed.kind = "package"
	default:
		return parsedObservation{}, false
	}
	if parsed.name == "" {
		parsed.name = parsed.displayName
	}
	return parsed, parsed.name != "" || len(parsed.executablePaths) > 0
}

func (definition product) match(observation parsedObservation) (float64, bool) {
	name := normalizeArtifact(observation.name)
	display := normalizeArtifact(observation.displayName)
	matched := false
	strength := 0.0
	switch observation.kind {
	case "process":
		if matchesAny(definition.Processes, name) {
			matched, strength = true, 0.82
		}
	case "service":
		if matchesAny(definition.Services, name) || matchesAny(definition.Services, display) {
			matched, strength = true, 0.95
		}
	case "package":
		if matchesAny(definition.Packages, name) || matchesAny(definition.Packages, display) {
			for _, excluded := range definition.PackageExclude {
				if containsToken(name, excluded) {
					return 0, false
				}
			}
			matched, strength = true, 0.90
		}
	}
	for _, executable := range observation.executablePaths {
		base := executableBase(executable)
		if matchesAny(definition.Executables, base) && strength < 0.86 {
			matched, strength = true, 0.86
		}
		lowerPath := strings.ToLower(executable)
		for _, needle := range definition.PathContains {
			if strings.Contains(lowerPath, strings.ToLower(needle)) && strength < 0.84 {
				matched, strength = true, 0.84
			}
		}
	}
	return strength, matched
}

func (builder *detectionBuilder) add(
	observation parsedObservation,
	sourceAssetID string,
	strength float64,
) {
	builder.kinds[observation.kind] = true
	if strength > builder.strength {
		builder.strength = strength
	}
	if observation.version != "" {
		builder.versions[observation.version] = true
	}
	for _, executable := range observation.executablePaths {
		builder.executablePath[executable] = true
	}
	switch observation.kind {
	case "process":
		builder.processCount++
		builder.processNames[observation.name] = true
		builder.running = true
	case "service":
		builder.serviceNames[observation.name] = true
		builder.running = builder.running || observation.running
		builder.serviceState = builder.serviceState || observation.serviceState
		builder.serviceStopped = builder.serviceStopped || observation.serviceStopped
	case "package":
		builder.packageNames[observation.name] = true
	}
	evidenceKey := observation.kind + "\x00" + sourceAssetID
	if builder.seenEvidence[evidenceKey] {
		return
	}
	builder.seenEvidence[evidenceKey] = true
	if len(builder.evidence) < maxEvidence {
		builder.evidence = append(builder.evidence, Evidence{
			Kind: observation.kind, Name: observation.name, SourceAssetID: sourceAssetID,
		})
	}
}

func (builder *detectionBuilder) finish() Detection {
	versions := sortedSet(builder.versions)
	sort.SliceStable(versions, func(left, right int) bool {
		return naturalVersionCompare(versions[left], versions[right]) < 0
	})
	version := ""
	if len(versions) > 0 {
		version = versions[len(versions)-1]
	}
	installState := "observed"
	if builder.kinds["package"] || builder.kinds["service"] {
		installState = "installed"
	}
	runtimeState := "unknown"
	switch {
	case builder.running:
		runtimeState = "running"
	case builder.kinds["service"] && builder.serviceState && builder.serviceStopped:
		runtimeState = "stopped"
	case builder.kinds["service"]:
		runtimeState = "unknown"
	}
	confidence := builder.strength
	if len(builder.kinds) >= 2 {
		confidence += 0.03
	}
	if len(builder.kinds) == 3 {
		confidence += 0.01
	}
	if confidence > 0.99 {
		confidence = 0.99
	}
	sort.Slice(builder.evidence, func(left, right int) bool {
		if builder.evidence[left].Kind != builder.evidence[right].Kind {
			return builder.evidence[left].Kind < builder.evidence[right].Kind
		}
		if builder.evidence[left].Name != builder.evidence[right].Name {
			return builder.evidence[left].Name < builder.evidence[right].Name
		}
		return builder.evidence[left].SourceAssetID < builder.evidence[right].SourceAssetID
	})
	return Detection{
		ProductKey: builder.product.Key, ProductName: builder.product.Name,
		Role: builder.product.Role, Vendor: builder.product.Vendor,
		Version: version, Versions: versions, InstallState: installState,
		RuntimeState: runtimeState, ServiceNames: sortedSet(builder.serviceNames),
		ProcessNames: sortedSet(builder.processNames), PackageNames: sortedSet(builder.packageNames),
		ExecutablePath: sortedSet(builder.executablePath), Evidence: builder.evidence,
		EvidenceCount: len(builder.seenEvidence), ProcessCount: builder.processCount,
		Detection: "builtin_catalog", CatalogVersion: CatalogVersion,
		Confidence: confidence,
	}
}

// naturalVersionCompare orders numeric runs by their numeric magnitude rather
// than their text bytes. Inventory often contains side-by-side major versions;
// lexical sorting would incorrectly present 9.4 as newer than 16.0 and 1.9 as
// newer than 1.10. Non-numeric runs remain deterministic without claiming full
// SemVer support for vendor-specific strings such as 23H2.
func naturalVersionCompare(left, right string) int {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	for leftIndex, rightIndex := 0, 0; ; {
		if leftIndex >= len(left) || rightIndex >= len(right) {
			switch {
			case leftIndex >= len(left) && rightIndex >= len(right):
				return 0
			case leftIndex >= len(left):
				return -1
			default:
				return 1
			}
		}
		leftDigit := isASCIIDigit(left[leftIndex])
		rightDigit := isASCIIDigit(right[rightIndex])
		if leftDigit && rightDigit {
			leftEnd, rightEnd := leftIndex, rightIndex
			for leftEnd < len(left) && isASCIIDigit(left[leftEnd]) {
				leftEnd++
			}
			for rightEnd < len(right) && isASCIIDigit(right[rightEnd]) {
				rightEnd++
			}
			leftNumber := strings.TrimLeft(left[leftIndex:leftEnd], "0")
			rightNumber := strings.TrimLeft(right[rightIndex:rightEnd], "0")
			if leftNumber == "" {
				leftNumber = "0"
			}
			if rightNumber == "" {
				rightNumber = "0"
			}
			if len(leftNumber) != len(rightNumber) {
				if len(leftNumber) < len(rightNumber) {
					return -1
				}
				return 1
			}
			if leftNumber != rightNumber {
				if leftNumber < rightNumber {
					return -1
				}
				return 1
			}
			leftIndex, rightIndex = leftEnd, rightEnd
			continue
		}
		if left[leftIndex] != right[rightIndex] {
			if left[leftIndex] < right[rightIndex] {
				return -1
			}
			return 1
		}
		leftIndex++
		rightIndex++
	}
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func serviceRuntime(payload map[string]any) (running, known, stopped bool) {
	if active, ok := payload["active"].(bool); ok {
		return active, true, !active
	}
	for _, key := range []string{"state", "active_state", "sub_state"} {
		value, _ := payload[key].(string)
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "running", "active":
			return true, true, false
		case "stopped", "inactive", "failed", "dead", "exited", "paused":
			known, stopped = true, true
		}
	}
	return false, known, stopped
}

// executablePaths strips service arguments. Persisting an ImagePath or command
// line verbatim is unsafe because command arguments often contain passwords.
func executablePaths(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	value = strings.TrimSuffix(value, " (deleted)")
	if strings.HasPrefix(value, `"`) {
		if end := strings.Index(value[1:], `"`); end >= 0 {
			value = value[1 : end+1]
		}
	} else {
		lower := strings.ToLower(value)
		if end := strings.Index(lower, ".exe"); end >= 0 {
			value = value[:end+4]
		} else if fields := strings.Fields(value); len(fields) > 0 {
			value = fields[0]
		}
	}
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" {
		return nil
	}
	return []string{value}
}

func executableBase(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if index := strings.LastIndex(normalized, "/"); index >= 0 {
		normalized = normalized[index+1:]
	}
	return normalizeArtifact(normalized)
}

func normalizeArtifact(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.Trim(normalized, `"'`)
	for _, suffix := range []string{".service", ".exe"} {
		normalized = strings.TrimSuffix(normalized, suffix)
	}
	return normalized
}

func matchesAny(patterns []string, value string) bool {
	if value == "" {
		return false
	}
	for _, pattern := range patterns {
		if glob(strings.ToLower(pattern), value) {
			return true
		}
	}
	return false
}

// glob supports only '*'. Catalogue entries are trusted constants; keeping the
// matcher this small avoids a regex engine in the ingest hot path.
func glob(pattern, value string) bool {
	if pattern == "" {
		return false
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}
	position := 0
	for index, part := range parts {
		if part == "" {
			continue
		}
		found := strings.Index(value[position:], part)
		if found < 0 || (index == 0 && found != 0) {
			return false
		}
		position += found + len(part)
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(value, last)
}

func containsToken(value, token string) bool {
	split := func(input string) []string {
		return strings.FieldsFunc(strings.ToLower(input), func(char rune) bool {
			return char == '-' || char == '_' || char == '.' || char == ' ' ||
				char == '/' || char == '\\' || char == '(' || char == ')' ||
				char == '[' || char == ']' || char == ',' || char == ':'
		})
	}
	fields := split(value)
	needleFields := split(token)
	if len(needleFields) == 0 {
		return false
	}
	valuePhrase := " " + strings.Join(fields, " ") + " "
	needlePhrase := " " + strings.Join(needleFields, " ") + " "
	return strings.Contains(valuePhrase, needlePhrase)
}

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
