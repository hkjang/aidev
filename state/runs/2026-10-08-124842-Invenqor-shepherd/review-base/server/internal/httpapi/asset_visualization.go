package httpapi

import (
	"sync"

	"context"
	"database/sql"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"strings"
	"time"
)

// The console must never pull the inventory into the browser to draw a chart, so
// every view is answered by one aggregation request. Each block below is shaped
// for the form that reads it, which is also why the API returns matrices and
// buckets rather than rows: the layout maths belongs on the client, the counting
// belongs in the database.

type visualizationCell struct {
	Row    string `json:"row"`
	Column string `json:"column"`
	Count  int64  `json:"count"`
	Stale  int64  `json:"stale"`
}

type visualizationMatrix struct {
	Rows    []string            `json:"rows"`
	Columns []string            `json:"columns"`
	Cells   []visualizationCell `json:"cells"`
	Maximum int64               `json:"maximum"`
}

type visualizationBucket struct {
	Label    string `json:"label"`
	MaxHours int    `json:"max_hours"`
	Count    int64  `json:"count"`
}

type visualizationBranch struct {
	Label    string            `json:"label"`
	Count    int64             `json:"count"`
	Children []statisticBucket `json:"children"`
}

type visualizationFlow struct {
	Date    string `json:"date"`
	Added   int64  `json:"added"`
	Removed int64  `json:"removed"`
	Total   int64  `json:"total"`
}

type visualizationNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Environment string `json:"environment"`
	Criticality string `json:"criticality"`
	Degree      int    `json:"degree"`
	Stale       bool   `json:"stale"`
}

type visualizationEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type visualizationGraph struct {
	Nodes     []visualizationNode `json:"nodes"`
	Edges     []visualizationEdge `json:"edges"`
	Truncated bool                `json:"truncated"`
	Total     int64               `json:"total_relations"`
}

// assetVisualizationFilter is the one slice every block is counted against, so
// the views can never disagree with each other or with the asset list.
type assetVisualizationFilter struct {
	Days        int
	Environment string
	Criticality string
	Type        string
	StaleHours  int
}

func (filter assetVisualizationFilter) where(
	arguments *[]any,
	alias string,
) string {
	clause := " AND " + alias + "deleted_at IS NULL"
	for column, value := range map[string]string{
		"environment": filter.Environment,
		"criticality": filter.Criticality,
		"type":        filter.Type,
	} {
		if strings.TrimSpace(value) == "" {
			continue
		}
		*arguments = append(*arguments, value)
		clause += fmt.Sprintf(
			" AND COALESCE(NULLIF(%s%s,''),'unknown') = $%d",
			alias, column, len(*arguments),
		)
	}
	return clause
}

const visualizationGraphNodeLimit = 160

func (s *Server) assetVisualization(
	response http.ResponseWriter,
	request *http.Request,
) {
	filter := assetVisualizationFilter{
		Days:        queryInt(request, "days", 30, 1, 365),
		Environment: strings.TrimSpace(request.URL.Query().Get("environment")),
		Criticality: strings.TrimSpace(request.URL.Query().Get("criticality")),
		Type:        strings.TrimSpace(request.URL.Query().Get("type")),
		StaleHours:  queryInt(request, "stale_hours", 24, 1, 24*90),
	}
	ctx := request.Context()
	now := time.Now().UTC()
	database := s.database.DB()

	total, err := s.visualizationTotal(ctx, filter)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	// Seven independent scans of the whole table, which used to run one after
	// another. They are read-only and share nothing, and PostgreSQL serves them
	// in parallel from shared buffers: measured on 500,000 assets, seven of
	// these take 319ms in sequence and 99ms together.
	//
	// The pool holds 25 connections and the rest of this handler needs some, so
	// this takes at most four at a time rather than all seven. A visualization
	// request must not be able to starve every other request on the Server.
	dimensions := map[string][]statisticBucket{}
	dimensionColumns := map[string]string{
		"type":             "type",
		"status":           "status",
		"environment":      "environment",
		"criticality":      "criticality",
		"source":           "source",
		"owner_department": "owner_department",
		"location":         "location",
	}
	var dimensionMutex sync.Mutex
	group, groupContext := errgroup.WithContext(ctx)
	group.SetLimit(4)
	for key, column := range dimensionColumns {
		key, column := key, column
		group.Go(func() error {
			arguments := []any{}
			buckets, err := groupedCounts(
				groupContext,
				database,
				`SELECT COALESCE(NULLIF(`+column+`,''),'unknown') AS label, COUNT(*)
				   FROM assets WHERE 1=1`+filter.where(&arguments, "")+`
				  GROUP BY label ORDER BY COUNT(*) DESC, label LIMIT 24`,
				arguments...,
			)
			if err != nil {
				return err
			}
			dimensionMutex.Lock()
			defer dimensionMutex.Unlock()
			dimensions[key] = buckets
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		s.internalError(response, request, err)
		return
	}

	// Left sequential deliberately. Running these four together as well was
	// measured and made the page slower - 1.18s to 1.40s on 500,000 assets -
	// because the parallel scans then contend rather than overlap. The seven
	// above are uniform and cheap enough to overlap; these are not.
	matrix, err := s.visualizationMatrix(ctx, filter, now)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	freshness, err := s.visualizationFreshness(ctx, filter, now)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	hierarchy, err := s.visualizationHierarchy(ctx, filter)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	flow, err := s.visualizationFlow(ctx, filter, now, total)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	graph, err := s.visualizationGraph(ctx, filter, now)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	stale, err := s.visualizationStale(ctx, filter, now)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	unowned, err := s.visualizationUnowned(ctx, filter)
	if err != nil {
		s.internalError(response, request, err)
		return
	}

	writeJSON(response, http.StatusOK, map[string]any{
		"generated_at": now,
		"window_days":  filter.Days,
		"stale_hours":  filter.StaleHours,
		"filter": map[string]any{
			"environment": filter.Environment,
			"criticality": filter.Criticality,
			"type":        filter.Type,
		},
		"totals": map[string]any{
			"assets":  total,
			"stale":   stale,
			"fresh":   total - stale,
			"unowned": unowned,
		},
		"dimensions": dimensions,
		"matrix":     matrix,
		"freshness":  freshness,
		"hierarchy":  hierarchy,
		"flow":       flow,
		"graph":      graph,
	})
}

func (s *Server) visualizationTotal(
	ctx context.Context,
	filter assetVisualizationFilter,
) (int64, error) {
	arguments := []any{}
	return scalarCount(
		ctx,
		s.database.DB(),
		`SELECT COUNT(*) FROM assets WHERE 1=1`+filter.where(&arguments, ""),
		arguments...,
	)
}

func (s *Server) visualizationStale(
	ctx context.Context,
	filter assetVisualizationFilter,
	now time.Time,
) (int64, error) {
	arguments := []any{now.Add(-time.Duration(filter.StaleHours) * time.Hour)}
	return scalarCount(
		ctx,
		s.database.DB(),
		`SELECT COUNT(*) FROM assets
		  WHERE last_seen_at < $1`+filter.where(&arguments, ""),
		arguments...,
	)
}

func (s *Server) visualizationUnowned(
	ctx context.Context,
	filter assetVisualizationFilter,
) (int64, error) {
	arguments := []any{}
	return scalarCount(
		ctx,
		s.database.DB(),
		`SELECT COUNT(*) FROM assets
		  WHERE TRIM(owner_department) = ''
		    AND owner_user_id IS NULL`+filter.where(&arguments, ""),
		arguments...,
	)
}

// visualizationMatrix answers "which business-critical assets sit in which
// environment, and how many of them have gone quiet" - the pairing an operator
// acts on, which no single-dimension breakdown can show.
func (s *Server) visualizationMatrix(
	ctx context.Context,
	filter assetVisualizationFilter,
	now time.Time,
) (visualizationMatrix, error) {
	arguments := []any{now.Add(-time.Duration(filter.StaleHours) * time.Hour)}
	rows, err := s.database.DB().QueryContext(
		ctx,
		`SELECT COALESCE(NULLIF(criticality,''),'unknown') AS row_label,
		        COALESCE(NULLIF(environment,''),'unknown') AS column_label,
		        COUNT(*),
		        SUM(CASE WHEN last_seen_at < $1 THEN 1 ELSE 0 END)
		   FROM assets WHERE 1=1`+filter.where(&arguments, "")+`
		  GROUP BY row_label, column_label`,
		arguments...,
	)
	if err != nil {
		return visualizationMatrix{}, fmt.Errorf("aggregate asset matrix: %w", err)
	}
	defer rows.Close()
	matrix := visualizationMatrix{
		Rows:    make([]string, 0),
		Columns: make([]string, 0),
		Cells:   make([]visualizationCell, 0),
	}
	seenRows := map[string]bool{}
	seenColumns := map[string]bool{}
	for rows.Next() {
		var cell visualizationCell
		var stale sql.NullInt64
		if err := rows.Scan(
			&cell.Row, &cell.Column, &cell.Count, &stale,
		); err != nil {
			return visualizationMatrix{}, err
		}
		cell.Stale = stale.Int64
		if cell.Count > matrix.Maximum {
			matrix.Maximum = cell.Count
		}
		if !seenRows[cell.Row] {
			seenRows[cell.Row] = true
			matrix.Rows = append(matrix.Rows, cell.Row)
		}
		if !seenColumns[cell.Column] {
			seenColumns[cell.Column] = true
			matrix.Columns = append(matrix.Columns, cell.Column)
		}
		matrix.Cells = append(matrix.Cells, cell)
	}
	if err := rows.Err(); err != nil {
		return visualizationMatrix{}, err
	}
	// Criticality is an ordered scale, so the axis follows severity rather than
	// the row count; anything unrecognised keeps a stable alphabetical tail.
	matrix.Rows = orderByPreference(matrix.Rows, criticalityOrder)
	matrix.Columns = orderByPreference(matrix.Columns, environmentOrder)
	return matrix, nil
}

var criticalityOrder = []string{
	"critical", "high", "medium", "normal", "low", "unknown",
}

var environmentOrder = []string{
	"production", "staging", "qa", "test", "development", "dr", "other",
	"unknown",
}

func orderByPreference(values []string, preference []string) []string {
	rank := map[string]int{}
	for index, value := range preference {
		rank[value] = index
	}
	ordered := make([]string, 0, len(values))
	for _, value := range preference {
		for _, candidate := range values {
			if candidate == value {
				ordered = append(ordered, candidate)
			}
		}
	}
	tail := make([]string, 0, len(values))
	for _, candidate := range values {
		if _, known := rank[candidate]; !known {
			tail = append(tail, candidate)
		}
	}
	sortStrings(tail)
	return append(ordered, tail...)
}

func sortStrings(values []string) {
	for index := 1; index < len(values); index++ {
		for position := index; position > 0 && values[position] < values[position-1]; position-- {
			values[position], values[position-1] = values[position-1], values[position]
		}
	}
}

// visualizationFreshness buckets by age instead of returning a fresh/stale
// boolean: "last seen 40 days ago" and "last seen 25 hours ago" call for very
// different actions, and a two-state indicator hides that difference.
func (s *Server) visualizationFreshness(
	ctx context.Context,
	filter assetVisualizationFilter,
	now time.Time,
) ([]visualizationBucket, error) {
	definitions := []visualizationBucket{
		{Label: "24시간 이내", MaxHours: 24},
		{Label: "7일 이내", MaxHours: 24 * 7},
		{Label: "30일 이내", MaxHours: 24 * 30},
		{Label: "90일 이내", MaxHours: 24 * 90},
		{Label: "90일 초과", MaxHours: 0},
	}
	previous := 0
	for index := range definitions {
		arguments := []any{}
		condition := ""
		if definitions[index].MaxHours > 0 {
			arguments = append(
				arguments,
				now.Add(-time.Duration(definitions[index].MaxHours)*time.Hour),
			)
			condition = " AND last_seen_at >= $1"
			if previous > 0 {
				arguments = append(
					arguments,
					now.Add(-time.Duration(previous)*time.Hour),
				)
				condition += " AND last_seen_at < $2"
			}
			previous = definitions[index].MaxHours
		} else {
			arguments = append(
				arguments,
				now.Add(-time.Duration(previous)*time.Hour),
			)
			condition = " AND last_seen_at < $1"
		}
		count, err := scalarCount(
			ctx,
			s.database.DB(),
			`SELECT COUNT(*) FROM assets WHERE 1=1`+condition+
				filter.where(&arguments, ""),
			arguments...,
		)
		if err != nil {
			return nil, fmt.Errorf("aggregate asset freshness: %w", err)
		}
		definitions[index].Count = count
	}
	return definitions, nil
}

// visualizationHierarchy feeds the treemap: environment is the outer grouping an
// operator thinks in, asset type is what fills it.
func (s *Server) visualizationHierarchy(
	ctx context.Context,
	filter assetVisualizationFilter,
) ([]visualizationBranch, error) {
	arguments := []any{}
	rows, err := s.database.DB().QueryContext(
		ctx,
		`SELECT COALESCE(NULLIF(environment,''),'unknown') AS branch,
		        COALESCE(NULLIF(type,''),'unknown') AS leaf,
		        COUNT(*)
		   FROM assets WHERE 1=1`+filter.where(&arguments, "")+`
		  GROUP BY branch, leaf
		  ORDER BY branch, COUNT(*) DESC, leaf`,
		arguments...,
	)
	if err != nil {
		return nil, fmt.Errorf("aggregate asset hierarchy: %w", err)
	}
	defer rows.Close()
	branches := make([]visualizationBranch, 0)
	index := map[string]int{}
	for rows.Next() {
		var branch, leaf string
		var count int64
		if err := rows.Scan(&branch, &leaf, &count); err != nil {
			return nil, err
		}
		position, found := index[branch]
		if !found {
			branches = append(branches, visualizationBranch{
				Label:    branch,
				Children: make([]statisticBucket, 0, 4),
			})
			position = len(branches) - 1
			index[branch] = position
		}
		branches[position].Count += count
		branches[position].Children = append(
			branches[position].Children,
			statisticBucket{Label: leaf, Count: count},
		)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for outer := 1; outer < len(branches); outer++ {
		for position := outer; position > 0 &&
			branches[position].Count > branches[position-1].Count; position-- {
			branches[position], branches[position-1] =
				branches[position-1], branches[position]
		}
	}
	return branches, nil
}

// visualizationFlow returns per-day arrivals and retirements plus the running
// total, so growth and churn can be read from the same request.
func (s *Server) visualizationFlow(
	ctx context.Context,
	filter assetVisualizationFilter,
	now time.Time,
	total int64,
) ([]visualizationFlow, error) {
	start := now.AddDate(0, 0, -(filter.Days - 1)).Truncate(24 * time.Hour)
	added, err := s.visualizationDailyCounts(
		ctx, filter, "first_seen_at", start,
	)
	if err != nil {
		return nil, err
	}
	removed, err := s.visualizationDailyCounts(
		ctx, filter, "deleted_at", start,
	)
	if err != nil {
		return nil, err
	}
	series := make([]visualizationFlow, 0, filter.Days)
	for day := filter.Days - 1; day >= 0; day-- {
		key := now.AddDate(0, 0, -day).Format("2006-01-02")
		series = append(series, visualizationFlow{
			Date:    key,
			Added:   added[key],
			Removed: removed[key],
		})
	}
	// Walk the running total backwards from today so the curve ends on the same
	// number the stat tiles report.
	running := total
	for index := len(series) - 1; index >= 0; index-- {
		series[index].Total = running
		running = running - series[index].Added + series[index].Removed
		if running < 0 {
			running = 0
		}
	}
	return series, nil
}

func (s *Server) visualizationDailyCounts(
	ctx context.Context,
	filter assetVisualizationFilter,
	column string,
	start time.Time,
) (map[string]int64, error) {
	arguments := []any{start}
	// deleted_at rows are exactly the ones the shared filter excludes, so the
	// retirement series carries its own predicate.
	scope := filter.where(&arguments, "")
	if column == "deleted_at" {
		scope = strings.Replace(scope, " AND deleted_at IS NULL", "", 1)
	}
	day := dateExpression(column)
	rows, err := s.database.DB().QueryContext(
		ctx,
		`SELECT `+day+`, COUNT(*) FROM assets
		  WHERE `+column+` >= $1`+scope+`
		  GROUP BY `+day,
		arguments...,
	)
	if err != nil {
		return nil, fmt.Errorf("aggregate asset %s counts: %w", column, err)
	}
	defer rows.Close()
	result := map[string]int64{}
	for rows.Next() {
		var date any
		var count int64
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		result[dateString(date)] = count
	}
	return result, rows.Err()
}

// visualizationGraph returns a bounded subgraph centred on the most connected
// assets. An unbounded topology is unreadable and an unbounded query is a
// denial of service, so the node cap is part of the contract and the response
// says when it truncated.
func (s *Server) visualizationGraph(
	ctx context.Context,
	filter assetVisualizationFilter,
	now time.Time,
) (visualizationGraph, error) {
	graph := visualizationGraph{
		Nodes: make([]visualizationNode, 0),
		Edges: make([]visualizationEdge, 0),
	}
	arguments := []any{}
	scope := filter.where(&arguments, "a.")
	relationTotal, err := scalarCount(
		ctx,
		s.database.DB(),
		`SELECT COUNT(*) FROM asset_relations r
		  WHERE r.valid_to IS NULL
		    AND EXISTS(SELECT 1 FROM assets a WHERE a.id = r.source_asset_id`+
			scope+`)`,
		arguments...,
	)
	if err != nil {
		return graph, fmt.Errorf("count asset relations: %w", err)
	}
	graph.Total = relationTotal

	arguments = []any{now.Add(-time.Duration(filter.StaleHours) * time.Hour)}
	scope = filter.where(&arguments, "a.")
	arguments = append(arguments, visualizationGraphNodeLimit)
	rows, err := s.database.DB().QueryContext(
		ctx,
		`SELECT a.id, a.name, COALESCE(NULLIF(a.type,''),'unknown'),
		        COALESCE(NULLIF(a.environment,''),'unknown'),
		        COALESCE(NULLIF(a.criticality,''),'unknown'),
		        CASE WHEN a.last_seen_at < $1 THEN 1 ELSE 0 END,
		        (SELECT COUNT(*) FROM asset_relations r
		          WHERE r.valid_to IS NULL
		            AND (r.source_asset_id = a.id OR r.target_asset_id = a.id)
		        ) AS degree
		   FROM assets a WHERE 1=1`+scope+`
		  ORDER BY degree DESC, a.name
		  LIMIT $`+fmt.Sprint(len(arguments)),
		arguments...,
	)
	if err != nil {
		return graph, fmt.Errorf("select topology nodes: %w", err)
	}
	defer rows.Close()
	included := map[string]bool{}
	for rows.Next() {
		var node visualizationNode
		var stale int
		if err := rows.Scan(
			&node.ID, &node.Name, &node.Type, &node.Environment,
			&node.Criticality, &stale, &node.Degree,
		); err != nil {
			return graph, err
		}
		node.Stale = stale == 1
		included[node.ID] = true
		graph.Nodes = append(graph.Nodes, node)
	}
	if err := rows.Err(); err != nil {
		return graph, err
	}
	if len(graph.Nodes) == 0 {
		return graph, nil
	}

	edgeRows, err := s.database.DB().QueryContext(
		ctx,
		`SELECT source_asset_id, target_asset_id, relation_type
		   FROM asset_relations WHERE valid_to IS NULL
		  ORDER BY relation_type LIMIT 2000`,
	)
	if err != nil {
		return graph, fmt.Errorf("select topology edges: %w", err)
	}
	defer edgeRows.Close()
	for edgeRows.Next() {
		var edge visualizationEdge
		if err := edgeRows.Scan(
			&edge.Source, &edge.Target, &edge.Type,
		); err != nil {
			return graph, err
		}
		// Only edges whose both ends survived the node cap can be drawn.
		if included[edge.Source] && included[edge.Target] {
			graph.Edges = append(graph.Edges, edge)
		}
	}
	if err := edgeRows.Err(); err != nil {
		return graph, err
	}
	graph.Truncated = int64(len(graph.Edges)) < relationTotal
	return graph, nil
}
