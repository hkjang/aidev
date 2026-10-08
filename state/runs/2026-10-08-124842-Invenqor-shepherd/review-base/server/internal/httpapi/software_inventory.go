package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hkjang/invenqor/server/internal/apitime"
	"github.com/hkjang/invenqor/server/internal/storage"
)

// softwareProductAttributes is the stable contract emitted by the automatic
// product detector.  Keeping this shape explicit prevents the console from
// having to understand collector-specific process, service and package payloads.
type softwareProductAttributes struct {
	ProductKey      string             `json:"product_key"`
	ProductName     string             `json:"product_name"`
	Role            string             `json:"role"`
	Vendor          string             `json:"vendor"`
	Version         string             `json:"version"`
	Versions        []string           `json:"versions"`
	InstallState    string             `json:"install_state"`
	RuntimeState    string             `json:"runtime_state"`
	ServiceNames    []string           `json:"service_names"`
	ProcessNames    []string           `json:"process_names"`
	PackageNames    []string           `json:"package_names"`
	ExecutablePaths []string           `json:"executable_paths"`
	Evidence        []softwareEvidence `json:"evidence"`
	DetectionMethod string             `json:"detection_method"`
	CatalogVersion  string             `json:"catalog_version"`
	EvidenceCount   int                `json:"evidence_count"`
	ProcessCount    int                `json:"process_count"`
	Confidence      float64            `json:"confidence"`
}

type softwareEvidence struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	SourceAssetID string `json:"source_asset_id,omitempty"`
}

type softwareHost struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type softwareProductItem struct {
	ID              string             `json:"id"`
	AssetKey        string             `json:"asset_key"`
	Status          string             `json:"status"`
	ProductKey      string             `json:"product_key"`
	ProductName     string             `json:"product_name"`
	Role            string             `json:"role"`
	Vendor          string             `json:"vendor"`
	Version         string             `json:"version"`
	Versions        []string           `json:"versions"`
	InstallState    string             `json:"install_state"`
	RuntimeState    string             `json:"runtime_state"`
	ServiceNames    []string           `json:"service_names"`
	ProcessNames    []string           `json:"process_names"`
	PackageNames    []string           `json:"package_names"`
	ExecutablePaths []string           `json:"executable_paths"`
	Evidence        []softwareEvidence `json:"evidence"`
	DetectionMethod string             `json:"detection_method"`
	CatalogVersion  string             `json:"catalog_version"`
	EvidenceCount   int                `json:"evidence_count"`
	ProcessCount    int                `json:"process_count"`
	Confidence      float64            `json:"confidence"`
	Host            softwareHost       `json:"host"`
	LastSeenAt      apitime.Time       `json:"last_seen_at"`
}

type softwareProductAggregate struct {
	ProductKey  string   `json:"product_key"`
	ProductName string   `json:"product_name"`
	Role        string   `json:"role"`
	Vendor      string   `json:"vendor"`
	Instances   int64    `json:"instances"`
	Hosts       int64    `json:"hosts"`
	Running     int64    `json:"running"`
	Versions    []string `json:"versions"`
}

type softwareInventorySummary struct {
	Products            int64                      `json:"products"`
	Instances           int64                      `json:"instances"`
	Hosts               int64                      `json:"hosts"`
	Running             int64                      `json:"running"`
	Stopped             int64                      `json:"stopped"`
	RuntimeUnknown      int64                      `json:"runtime_unknown"`
	Installed           int64                      `json:"installed"`
	ObservedOnly        int64                      `json:"observed_only"`
	HighConfidence      int64                      `json:"high_confidence"`
	NeedsReview         int64                      `json:"needs_review"`
	WithProcessEvidence int64                      `json:"with_process_evidence"`
	MappedProcesses     int64                      `json:"mapped_processes"`
	TopProducts         []softwareProductAggregate `json:"top_products"`
}

type softwareInventoryQuery struct {
	Query        string
	Role         string
	Vendor       string
	RuntimeState string
	Confidence   string
	Limit        int
	Offset       int
}

type softwareInventoryFilters struct {
	Roles   []string `json:"roles"`
	Vendors []string `json:"vendors"`
}

type softwareInventoryResult struct {
	Summary softwareInventorySummary `json:"summary"`
	Items   []softwareProductItem    `json:"items"`
	Total   int                      `json:"total"`
	Limit   int                      `json:"limit"`
	Offset  int                      `json:"offset"`
	HasMore bool                     `json:"has_more"`
	Filters softwareInventoryFilters `json:"filters"`
}

const activeSoftwareProjection = `
	FROM software_product_inventory p
	JOIN assets a ON a.id = p.asset_id
	WHERE a.type = 'software_product'
	  AND a.classification_source = 'software_catalog'
	  AND a.deleted_at IS NULL`

// listSoftwareProducts exposes product-shaped inventory rather than making the
// browser reconstruct it from thousands of raw process observations.  Every
// row remains host-scoped, while summary values de-duplicate product and host
// identities for an operations-friendly view.
func (s *Server) listSoftwareProducts(w http.ResponseWriter, r *http.Request) {
	result, err := s.querySoftwareInventory(r.Context(), softwareInventoryQuery{
		Query:        r.URL.Query().Get("q"),
		Role:         r.URL.Query().Get("role"),
		Vendor:       r.URL.Query().Get("vendor"),
		RuntimeState: r.URL.Query().Get("runtime_state"),
		Confidence:   r.URL.Query().Get("confidence"),
		Offset:       queryInt(r, "offset", 0, 0, 1_000_000),
		Limit:        queryInt(r, "limit", 50, 1, 200),
	})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) querySoftwareInventory(
	ctx context.Context,
	query softwareInventoryQuery,
) (softwareInventoryResult, error) {
	query.Query = strings.ToLower(strings.TrimSpace(query.Query))
	query.Role = strings.TrimSpace(query.Role)
	query.Vendor = strings.TrimSpace(query.Vendor)
	query.RuntimeState = strings.TrimSpace(query.RuntimeState)
	query.Confidence = strings.TrimSpace(query.Confidence)
	result := softwareInventoryResult{
		Items:  []softwareProductItem{},
		Limit:  query.Limit,
		Offset: query.Offset,
		Filters: softwareInventoryFilters{
			Roles:   []string{},
			Vendors: []string{},
		},
	}
	result.Summary.TopProducts = []softwareProductAggregate{}

	transaction, err := s.database.DB().BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return result, err
	}
	defer transaction.Rollback()

	if err := readSoftwareInventorySummary(ctx, transaction, &result.Summary); err != nil {
		return result, err
	}
	topProducts, err := readTopSoftwareProducts(ctx, transaction)
	if err != nil {
		return result, err
	}
	result.Summary.TopProducts = topProducts
	result.Filters.Roles, err = readSoftwareFacet(ctx, transaction, "role")
	if err != nil {
		return result, err
	}
	result.Filters.Vendors, err = readSoftwareFacet(ctx, transaction, "vendor")
	if err != nil {
		return result, err
	}

	arguments := make([]any, 0, 7)
	where := softwareInventoryWhere(query, &arguments)
	if err := transaction.QueryRowContext(
		ctx,
		"SELECT COUNT(*)"+activeSoftwareProjection+where,
		arguments...,
	).Scan(&result.Total); err != nil {
		return result, err
	}
	if result.Offset > result.Total {
		result.Offset = result.Total
	}

	pageArguments := append([]any(nil), arguments...)
	limitPlaceholder := appendSoftwareArgument(&pageArguments, result.Limit)
	offsetPlaceholder := appendSoftwareArgument(&pageArguments, result.Offset)
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT p.asset_id, a.asset_key, a.status,
		        p.product_key, p.product_name, p.role, p.vendor, p.version,
		        p.install_state, p.runtime_state, p.confidence,
		        p.process_count, p.evidence_count, p.catalog_version,
		        a.attributes_json, a.last_seen_at,
		        (SELECT CAST(h.id AS TEXT)
		           FROM asset_relations rel
		           JOIN assets h ON h.id = rel.target_asset_id
		          WHERE rel.source_asset_id = p.asset_id
		            AND rel.relation_type = 'runs_on'
		            AND rel.valid_to IS NULL
		            AND rel.status = 'active'
		          ORDER BY h.name, h.id LIMIT 1),
		        (SELECT h.name
		           FROM asset_relations rel
		           JOIN assets h ON h.id = rel.target_asset_id
		          WHERE rel.source_asset_id = p.asset_id
		            AND rel.relation_type = 'runs_on'
		            AND rel.valid_to IS NULL
		            AND rel.status = 'active'
		          ORDER BY h.name, h.id LIMIT 1)
		`+activeSoftwareProjection+where+`
		ORDER BY LOWER(p.product_name), p.asset_id
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder,
		pageArguments...,
	)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item softwareProductItem
		var rawAttributes string
		var hostID, hostName sql.NullString
		if err := rows.Scan(
			&item.ID, &item.AssetKey, &item.Status,
			&item.ProductKey, &item.ProductName, &item.Role, &item.Vendor,
			&item.Version, &item.InstallState, &item.RuntimeState,
			&item.Confidence, &item.ProcessCount, &item.EvidenceCount,
			&item.CatalogVersion, &rawAttributes, &item.LastSeenAt,
			&hostID, &hostName,
		); err != nil {
			rows.Close()
			return result, err
		}
		var attributes softwareProductAttributes
		_ = json.Unmarshal([]byte(rawAttributes), &attributes)
		item.Versions = nonNilStrings(attributes.Versions)
		item.ServiceNames = nonNilStrings(attributes.ServiceNames)
		item.ProcessNames = nonNilStrings(attributes.ProcessNames)
		item.PackageNames = nonNilStrings(attributes.PackageNames)
		item.ExecutablePaths = nonNilStrings(attributes.ExecutablePaths)
		item.Evidence = attributes.Evidence
		item.DetectionMethod = firstNonEmpty(attributes.DetectionMethod, "builtin_catalog")
		item.Host = softwareHost{ID: hostID.String, Name: hostName.String}
		if len(item.Evidence) == 0 {
			item.Evidence = inferredSoftwareEvidence(item)
		}
		if item.Evidence == nil {
			item.Evidence = []softwareEvidence{}
		}
		if item.EvidenceCount < len(item.Evidence) {
			item.EvidenceCount = len(item.Evidence)
		}
		if item.ProcessCount < len(item.ProcessNames) {
			item.ProcessCount = len(item.ProcessNames)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	result.HasMore = result.Offset+len(result.Items) < result.Total
	if err := transaction.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func softwareInventoryWhere(query softwareInventoryQuery, arguments *[]any) string {
	var where strings.Builder
	if query.Query != "" {
		where.WriteString(" AND p.search_text LIKE ")
		where.WriteString(appendSoftwareArgument(
			arguments, storage.LikeContains(query.Query),
		))
		where.WriteString(storage.LikeEscapeClause)
	}
	for _, filter := range []struct {
		column string
		value  string
	}{
		{"role", query.Role},
		{"vendor", query.Vendor},
		{"runtime_state", query.RuntimeState},
	} {
		if filter.value == "" {
			continue
		}
		where.WriteString(" AND p.")
		where.WriteString(filter.column)
		where.WriteString(" = ")
		where.WriteString(appendSoftwareArgument(arguments, filter.value))
	}
	if query.Confidence == "high" {
		where.WriteString(" AND p.confidence >= 0.8")
	} else if query.Confidence == "review" {
		where.WriteString(" AND p.confidence < 0.8")
	}
	return where.String()
}

func appendSoftwareArgument(arguments *[]any, value any) string {
	*arguments = append(*arguments, value)
	return fmt.Sprintf("$%d", len(*arguments))
}

func readSoftwareInventorySummary(
	ctx context.Context,
	transaction *sql.Tx,
	summary *softwareInventorySummary,
) error {
	return transaction.QueryRowContext(
		ctx,
		`SELECT COUNT(DISTINCT p.product_key), COUNT(*),
		        COUNT(DISTINCT p.agent_id),
		        COALESCE(SUM(CASE WHEN p.runtime_state = 'running' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN p.runtime_state = 'stopped' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN p.runtime_state NOT IN ('running','stopped') THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN p.install_state = 'installed' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN p.install_state = 'observed' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN p.confidence >= 0.8 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN p.confidence < 0.8 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN p.process_count > 0 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(p.process_count), 0)`+activeSoftwareProjection,
	).Scan(
		&summary.Products,
		&summary.Instances,
		&summary.Hosts,
		&summary.Running,
		&summary.Stopped,
		&summary.RuntimeUnknown,
		&summary.Installed,
		&summary.ObservedOnly,
		&summary.HighConfidence,
		&summary.NeedsReview,
		&summary.WithProcessEvidence,
		&summary.MappedProcesses,
	)
}

func readTopSoftwareProducts(
	ctx context.Context,
	transaction *sql.Tx,
) ([]softwareProductAggregate, error) {
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT p.product_key, MIN(p.product_name), MIN(p.role), MIN(p.vendor),
		        COUNT(*), COUNT(DISTINCT p.agent_id),
		        COALESCE(SUM(CASE WHEN p.runtime_state = 'running' THEN 1 ELSE 0 END), 0)`+
			activeSoftwareProjection+`
		GROUP BY p.product_key
		ORDER BY COUNT(*) DESC, MIN(p.product_name), p.product_key
		LIMIT 10`,
	)
	if err != nil {
		return nil, err
	}
	top := make([]softwareProductAggregate, 0, 10)
	for rows.Next() {
		var item softwareProductAggregate
		if err := rows.Scan(
			&item.ProductKey,
			&item.ProductName,
			&item.Role,
			&item.Vendor,
			&item.Instances,
			&item.Hosts,
			&item.Running,
		); err != nil {
			rows.Close()
			return nil, err
		}
		item.Versions = []string{}
		top = append(top, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range top {
		versionRows, err := transaction.QueryContext(
			ctx,
			`SELECT DISTINCT p.version`+activeSoftwareProjection+`
			 AND p.product_key = $1 AND p.version <> ''
			 ORDER BY p.version`,
			top[index].ProductKey,
		)
		if err != nil {
			return nil, err
		}
		for versionRows.Next() {
			var version string
			if err := versionRows.Scan(&version); err != nil {
				versionRows.Close()
				return nil, err
			}
			top[index].Versions = append(top[index].Versions, version)
		}
		if err := versionRows.Err(); err != nil {
			versionRows.Close()
			return nil, err
		}
		if err := versionRows.Close(); err != nil {
			return nil, err
		}
	}
	return top, nil
}

func readSoftwareFacet(
	ctx context.Context,
	transaction *sql.Tx,
	column string,
) ([]string, error) {
	if column != "role" && column != "vendor" {
		return nil, fmt.Errorf("unsupported software facet %q", column)
	}
	rows, err := transaction.QueryContext(
		ctx,
		"SELECT DISTINCT p."+column+activeSoftwareProjection+
			" ORDER BY p."+column,
	)
	if err != nil {
		return nil, err
	}
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			rows.Close()
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return values, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func inferredSoftwareEvidence(item softwareProductItem) []softwareEvidence {
	result := make([]softwareEvidence, 0, len(item.ServiceNames)+len(item.ProcessNames)+len(item.PackageNames))
	for _, value := range item.ServiceNames {
		result = append(result, softwareEvidence{Kind: "service", Name: value})
	}
	for _, value := range item.ProcessNames {
		result = append(result, softwareEvidence{Kind: "process", Name: value})
	}
	for _, value := range item.PackageNames {
		result = append(result, softwareEvidence{Kind: "package", Name: value})
	}
	return result
}
