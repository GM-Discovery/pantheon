package pantheon

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultHumanMapRoot = "00 Pantheon Core/Weave/Human Maps"

const humanMapGeneratedMarker = "<!-- PANTHEON-HUMAN-MAP:GENERATED -->"

type humanMapCluster struct {
	ID          string
	Title       string
	Path        string
	AnchorTerms []string
	NodeIndices []int
}

type humanMapState struct {
	Version          string `json:"version"`
	RunID            string `json:"run_id"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	VaultPath        string `json:"vault_path"`
	GraphPath        string `json:"graph_path"`
	MapRoot          string `json:"map_root"`
	BatchSize        int    `json:"batch_size"`
	NextOffset       int    `json:"next_offset"`
	TotalMaps        int    `json:"total_maps"`
	CompletedBatches int    `json:"completed_batches"`
	InventoryHash    string `json:"inventory_hash"`
	Status           string `json:"status"`
}

type humanMapTermScore struct {
	Term  string
	Score float64
}

func loadHumanMapState(path string) (humanMapState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return humanMapState{}, err
	}
	var state humanMapState
	if err := json.Unmarshal(data, &state); err != nil {
		return humanMapState{}, err
	}
	if state.Version != chunkLinkGraphVersion || (state.Status != "active" && state.Status != "completed") {
		return humanMapState{}, errors.New("invalid human map state")
	}
	if state.NextOffset < 0 || state.NextOffset > state.TotalMaps {
		return humanMapState{}, errors.New("invalid human map cursor")
	}
	return state, nil
}

func humanMapSlug(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(strings.TrimSuffix(b.String(), "-"), "-")
}

func humanMapTermTitle(term string) string {
	term = strings.ReplaceAll(strings.ReplaceAll(term, "_", " "), "-", " ")
	if term == "" {
		return "Unclassified evidence"
	}
	return strings.ToUpper(term[:1]) + term[1:]
}

func chooseHumanMapAnchors(graph chunkLinkGraph, mapCount int) []string {
	df := map[string]int{}
	for _, node := range graph.Nodes {
		for _, term := range node.Terms {
			df[term]++
		}
	}
	total := len(graph.Nodes)
	candidates := make([]humanMapTermScore, 0, len(df))
	for term, count := range df {
		if len(term) < 5 || count < 8 || count > total/3 {
			continue
		}
		// Prefer terms common enough to form a cluster but uncommon enough to
		// distinguish a topic from the whole archive.
		score := math.Log1p(float64(count)) * math.Log(float64(total+1)/float64(count+1))
		candidates = append(candidates, humanMapTermScore{Term: term, Score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Term < candidates[j].Term
	})
	if mapCount > len(candidates) {
		mapCount = len(candidates)
	}
	anchors := make([]string, 0, mapCount)
	for _, candidate := range candidates[:mapCount] {
		anchors = append(anchors, candidate.Term)
	}
	return anchors
}

func buildHumanMapPlan(graph chunkLinkGraph, mapCount int, root string) []humanMapCluster {
	type sourceInfo struct {
		path  string
		title string
		terms map[string]bool
		nodes []int
	}
	sourcesByPath := map[string]*sourceInfo{}
	for i, node := range graph.Nodes {
		info := sourcesByPath[node.Path]
		if info == nil {
			title := strings.TrimSuffix(filepath.Base(node.Path), filepath.Ext(node.Path))
			info = &sourceInfo{path: node.Path, title: title, terms: map[string]bool{}}
			for _, term := range chunkLinkTerms(title) {
				info.terms[term] = true
			}
			sourcesByPath[node.Path] = info
		}
		info.nodes = append(info.nodes, i)
	}
	sources := make([]*sourceInfo, 0, len(sourcesByPath))
	for _, info := range sourcesByPath {
		sources = append(sources, info)
	}
	sort.SliceStable(sources, func(i, j int) bool {
		if len(sources[i].nodes) != len(sources[j].nodes) {
			return len(sources[i].nodes) > len(sources[j].nodes)
		}
		return sources[i].title < sources[j].title
	})
	if mapCount > len(sources) {
		mapCount = len(sources)
	}
	if mapCount == 0 {
		return nil
	}
	clusters := make([]humanMapCluster, mapCount)
	clusterTerms := make([]map[string]bool, mapCount)
	for i := 0; i < mapCount; i++ {
		seed := sources[i]
		clusters[i] = humanMapCluster{ID: fmt.Sprintf("map-%03d", i+1), Title: seed.title, AnchorTerms: []string{seed.title}, NodeIndices: append([]int(nil), seed.nodes...)}
		clusterTerms[i] = seed.terms
	}
	for _, source := range sources[mapCount:] {
		best := 0
		bestScore := -1
		for i := range clusters {
			score := 0
			for term := range source.terms {
				if clusterTerms[i][term] {
					score++
				}
			}
			if score > bestScore || score == bestScore && len(clusters[i].NodeIndices) < len(clusters[best].NodeIndices) {
				best, bestScore = i, score
			}
		}
		clusters[best].NodeIndices = append(clusters[best].NodeIndices, source.nodes...)
		for term := range source.terms {
			clusterTerms[best][term] = true
		}
	}
	for i := range clusters {
		clusters[i].Path = filepath.ToSlash(filepath.Join(root, fmt.Sprintf("Topic Map %03d — %s.md", i+1, humanMapSlug(clusters[i].Title))))
	}
	return clusters
}

func renderHumanMapPage(graph chunkLinkGraph, cluster humanMapCluster, root string) []byte {
	degree := map[string]float64{}
	for _, edge := range graph.Edges {
		degree[edge.SourceID] += edge.Score
		degree[edge.TargetID] += edge.Score
	}
	nodes := append([]int(nil), cluster.NodeIndices...)
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := graph.Nodes[nodes[i]], graph.Nodes[nodes[j]]
		if degree[a.ID] != degree[b.ID] {
			return degree[a.ID] > degree[b.ID]
		}
		return a.ID < b.ID
	})
	if len(nodes) > 12 {
		nodes = nodes[:12]
	}
	sources := map[string]string{}
	for _, nodeIndex := range cluster.NodeIndices {
		node := graph.Nodes[nodeIndex]
		sources[node.Path] = strings.TrimSuffix(filepath.Base(node.Path), filepath.Ext(node.Path))
	}
	sourcePaths := make([]string, 0, len(sources))
	for path := range sources {
		sourcePaths = append(sourcePaths, path)
	}
	sort.Strings(sourcePaths)
	if len(sourcePaths) > 10 {
		sourcePaths = sourcePaths[:10]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\npantheon_human_map: true\nmap_id: %s\nmap_title: %q\nshard_count: %d\nsource_count: %d\n---\n\n%s\n# Topic map — %s\n\nThis is a human-facing map of user-authored evidence. It is derived from the raw lexical graph; it is not a canonical claim set.\n\n**%d evidence shards** from **%d source conversations**.\n\n## Key concepts\n\n", cluster.ID, cluster.Title, len(cluster.NodeIndices), len(sources), humanMapGeneratedMarker, cluster.Title, len(cluster.NodeIndices), len(sources))
	for _, term := range cluster.AnchorTerms {
		fmt.Fprintf(&b, "- `%s`\n", term)
	}
	b.WriteString("\n## Representative evidence\n\n")
	for i, nodeIndex := range nodes {
		node := graph.Nodes[nodeIndex]
		title := strings.TrimSuffix(filepath.Base(node.Path), filepath.Ext(node.Path))
		excerpt := strings.TrimSpace(node.Excerpt)
		if excerpt == "" || strings.HasPrefix(excerpt, "{") || strings.HasPrefix(excerpt, "[") || strings.Contains(excerpt, `"type"`) {
			excerpt = "user-authored evidence shard; open the link to inspect the preserved text"
		}
		if len(excerpt) > 180 {
			excerpt = excerpt[:177] + "..."
		}
		fmt.Fprintf(&b, "%d. [[%s|%s — evidence %02d]] — %s\n", i+1, userShardRelativePath(root, node.ID), title, i+1, excerpt)
	}
	b.WriteString("\n## Source conversations\n\n")
	for _, path := range sourcePaths {
		fmt.Fprintf(&b, "- [[%s|%s]]\n", path, sources[path])
	}
	b.WriteString("\n> Raw shards and the complete lexical graph remain preserved under the user-shard layer. This map is the readable entry point.\n")
	return []byte(b.String())
}

func renderHumanMapIndex(graph chunkLinkGraph, clusters []humanMapCluster, completed int, status string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "---\npantheon_human_map_index: true\nmap_count: %d\ncompleted_maps: %d\nstatus: %s\nshard_count: %d\ninventory_hash: %s\n---\n\n# User Knowledge Map\n\nThis is the human-facing entry point for the user-authored archive. It deliberately links to named topic maps instead of opening the 14,745-node raw shard graph.\n\n**%d topic maps** cover **%d user-authored shards**. Agent-authored chunks are excluded. Open this note's local graph at depth 1 for the readable map of topics.\n\n", len(clusters), completed, status, len(graph.Nodes), graph.InventoryHash, len(clusters), len(graph.Nodes))
	for i := 0; i < completed && i < len(clusters); i++ {
		cluster := clusters[i]
		fmt.Fprintf(&b, "- [[%s|%s]] — %d evidence shards\n", cluster.Path, cluster.Title, len(cluster.NodeIndices))
	}
	if completed < len(clusters) {
		fmt.Fprintf(&b, "\nProgress: %d/%d topic maps generated. Resume with `pantheon weave-links build-human-map`.\n", completed, len(clusters))
	} else {
		b.WriteString("\nThe raw shard graph remains available for drill-down, but this map is the default human-readable view.\n")
	}
	return []byte(b.String())
}

func runHumanMapBuilder(statePath, graphPath, vault, root string, mapCount, batchSize, batches int) error {
	if mapCount <= 0 || mapCount > 500 || batchSize <= 0 || batchSize > 20 || batches <= 0 || batches > 1000 {
		return errors.New("human map settings are out of range")
	}
	graph, err := loadChunkLinkGraph(graphPath)
	if err != nil {
		return err
	}
	if graph.VaultPath != vault || graph.Authorship != "nexus_user" {
		return errors.New("chunk graph is not the current user-authored graph for this Vault")
	}
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" || strings.HasPrefix(root, "../") || filepath.IsAbs(root) {
		return errors.New("human map root must be a non-empty Vault-relative directory")
	}
	clusters := buildHumanMapPlan(graph, mapCount, root)
	if len(clusters) == 0 {
		return errors.New("could not derive human map topics from the user graph")
	}
	state, err := loadHumanMapState(statePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		now := time.Now().UTC()
		state = humanMapState{Version: chunkLinkGraphVersion, RunID: now.Format("20060102T150405.000000000Z"), CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano), VaultPath: vault, GraphPath: graphPath, MapRoot: root, BatchSize: batchSize, TotalMaps: len(clusters), InventoryHash: graph.InventoryHash, Status: "active"}
		if err := writeJSONAtomic(statePath, state, 0o600); err != nil {
			return err
		}
	} else {
		if state.Status == "completed" {
			return fmt.Errorf("human map run %s is already complete; use a new --state path to rebuild", state.RunID)
		}
		if state.VaultPath != vault || state.GraphPath != graphPath || state.MapRoot != root || state.InventoryHash != graph.InventoryHash || state.TotalMaps != len(clusters) {
			return errors.New("human map inputs changed; start a new map state")
		}
		if state.BatchSize != batchSize {
			return fmt.Errorf("existing human map run uses batch size %d", state.BatchSize)
		}
	}
	outputDir := filepath.Join(vault, filepath.FromSlash(root))
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return err
	}
	indexPath := filepath.Join(outputDir, "User Knowledge Map.md")
	for batch := 0; batch < batches && state.Status == "active"; batch++ {
		end := min(state.NextOffset+state.BatchSize, len(clusters))
		for i := state.NextOffset; i < end; i++ {
			cluster := clusters[i]
			path := filepath.Join(vault, filepath.FromSlash(cluster.Path))
			content := renderHumanMapPage(graph, cluster, defaultUserShardRoot)
			before, readErr := readRegularFileNoSymlink(path)
			if readErr == nil && !strings.Contains(string(before), humanMapGeneratedMarker) {
				return fmt.Errorf("refusing to overwrite non-generated human map: %s", cluster.Path)
			}
			if err := writeBytesAtomic(path, content, 0o600); err != nil {
				return err
			}
		}
		state.NextOffset = end
		state.CompletedBatches++
		if state.NextOffset >= len(clusters) {
			state.Status = "completed"
		}
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := writeBytesAtomic(indexPath, renderHumanMapIndex(graph, clusters, state.NextOffset, state.Status), 0o600); err != nil {
			return err
		}
		if err := writeJSONAtomic(statePath, state, 0o600); err != nil {
			return err
		}
		fmt.Printf("Human map batch %d: %d/%d topic maps\n", state.CompletedBatches, state.NextOffset, len(clusters))
	}
	return nil
}
