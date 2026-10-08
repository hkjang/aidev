-- Query projection for the automatically detected software catalogue.  The
-- normalized asset JSON remains the source of truth. These scalar columns keep
-- filtering, counts and dashboards in the database instead of loading every
-- product into each API pod.
CREATE TABLE software_catalog_reconciliations (
    agent_id TEXT PRIMARY KEY REFERENCES agents(id) ON DELETE CASCADE,
    catalog_version TEXT NOT NULL DEFAULT '',
    reconciled_at TEXT
);

CREATE TABLE software_product_inventory (
    asset_id TEXT PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    product_key TEXT NOT NULL,
    product_name TEXT NOT NULL,
    role TEXT NOT NULL,
    vendor TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    install_state TEXT NOT NULL,
    runtime_state TEXT NOT NULL,
    confidence REAL NOT NULL,
    process_count INTEGER NOT NULL DEFAULT 0,
    evidence_count INTEGER NOT NULL DEFAULT 0,
    catalog_version TEXT NOT NULL DEFAULT '',
    search_text TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (agent_id, product_key)
);

CREATE INDEX software_product_inventory_name_idx
    ON software_product_inventory(LOWER(product_name), asset_id);
CREATE INDEX software_product_inventory_role_idx
    ON software_product_inventory(role, LOWER(product_name), asset_id);
CREATE INDEX software_product_inventory_vendor_idx
    ON software_product_inventory(vendor, LOWER(product_name), asset_id);
CREATE INDEX software_product_inventory_runtime_confidence_idx
    ON software_product_inventory(runtime_state, confidence, LOWER(product_name), asset_id);
CREATE INDEX software_product_inventory_product_version_idx
    ON software_product_inventory(product_key, version, agent_id);
CREATE INDEX software_product_inventory_updated_at_idx
    ON software_product_inventory(updated_at DESC);

-- Preserve products already inferred before this projection was introduced.
-- Later inventory events refresh the same rows through the reconciliation
-- transaction.
INSERT INTO software_product_inventory(
    asset_id, agent_id, product_key, product_name, role, vendor, version,
    install_state, runtime_state, confidence, process_count, evidence_count,
    catalog_version, search_text, updated_at
)
SELECT
    a.id,
    s.agent_id,
    COALESCE(NULLIF(json_extract(a.attributes_json, '$.product_key'), ''), s.source_asset_id),
    COALESCE(NULLIF(json_extract(a.attributes_json, '$.product_name'), ''), a.name),
    COALESCE(NULLIF(json_extract(a.attributes_json, '$.role'), ''), 'other'),
    COALESCE(NULLIF(json_extract(a.attributes_json, '$.vendor'), ''), 'unknown'),
    COALESCE(json_extract(a.attributes_json, '$.version'), ''),
    COALESCE(NULLIF(json_extract(a.attributes_json, '$.install_state'), ''), 'unknown'),
    COALESCE(NULLIF(json_extract(a.attributes_json, '$.runtime_state'), ''), 'unknown'),
    a.confidence,
    CAST(COALESCE(json_extract(a.attributes_json, '$.process_count'), 0) AS INTEGER),
    CAST(COALESCE(json_extract(a.attributes_json, '$.evidence_count'), 0) AS INTEGER),
    COALESCE(json_extract(a.attributes_json, '$.catalog_version'), ''),
    LOWER(
        s.source_asset_id || ' ' ||
        a.name || ' ' ||
        COALESCE(json_extract(a.attributes_json, '$.role'), '') || ' ' ||
        COALESCE(json_extract(a.attributes_json, '$.vendor'), '') || ' ' ||
        COALESCE(json_extract(a.attributes_json, '$.version'), '') || ' ' ||
        COALESCE((SELECT h.name
                    FROM asset_relations rel
                    JOIN assets h ON h.id = rel.target_asset_id
                   WHERE rel.source_asset_id = a.id
                     AND rel.relation_type = 'runs_on'
                     AND rel.valid_to IS NULL
                     AND rel.status = 'active'
                   ORDER BY h.name, h.id
                   LIMIT 1), '') || ' ' ||
        COALESCE(json_extract(a.attributes_json, '$.service_names'), '') || ' ' ||
        COALESCE(json_extract(a.attributes_json, '$.process_names'), '') || ' ' ||
        COALESCE(json_extract(a.attributes_json, '$.package_names'), '')
    ),
    a.updated_at
FROM assets a
JOIN asset_sources s ON s.asset_id = a.id
WHERE a.type = 'software_product'
  AND a.classification_source = 'software_catalog'
  AND a.deleted_at IS NULL
  AND json_valid(a.attributes_json)
  AND s.category = 'software.product'
  AND s.deleted_at IS NULL
  AND s.agent_id IS NOT NULL;
