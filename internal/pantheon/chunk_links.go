package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

const chunkLinkGraphVersion = "pantheon-user-chunk-link-graph-v1"

type chunkLinkNode struct {
	ID      string   `json:"id"`
	Path    string   `json:"path"`
	Ordinal int      `json:"ordinal"`
	Excerpt string   `json:"excerpt"`
	Terms   []string `json:"terms,omitempty"`
}

type chunkLinkEdge struct {
	ID            string   `json:"id"`
	SourceID      string   `json:"source_id"`
	TargetID      string   `json:"target_id"`
	SourcePath    string   `json:"source_path"`
	TargetPath    string   `json:"target_path"`
	SharedTerms   []string `json:"shared_terms"`
	Score         float64  `json:"score"`
	Relationship  string   `json:"relationship"`
	Assessment    string   `json:"assessment"`
	Uncertainty   string   `json:"uncertainty"`
	SourceExcerpt string   `json:"source_excerpt"`
	TargetExcerpt string   `json:"target_excerpt"`
}

type chunkLinkGraph struct {
	Version       string          `json:"version"`
	GeneratedAt   string          `json:"generated_at"`
	VaultPath     string          `json:"vault_path"`
	InventoryHash string          `json:"inventory_hash"`
	Authorship    string          `json:"authorship"`
	NodeCount     int             `json:"node_count"`
	EdgeCount     int             `json:"edge_count"`
	Nodes         []chunkLinkNode `json:"nodes"`
	Edges         []chunkLinkEdge `json:"edges"`
}

type chunkLinkState struct {
	Version          string `json:"version"`
	RunID            string `json:"run_id"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	VaultPath        string `json:"vault_path"`
	InventoryHash    string `json:"inventory_hash"`
	BatchSize        int    `json:"batch_size"`
	MaxLinks         int    `json:"max_links_per_chunk"`
	NextOffset       int    `json:"next_offset"`
	TotalChunks      int    `json:"total_chunks"`
	CompletedBatches int    `json:"completed_batches"`
	GraphPath        string `json:"graph_path"`
	Status           string `json:"status"`
}

type chunkLinkCorpus struct {
	Nodes             []chunkLinkNode
	Terms             [][]string
	Postings          map[string][]int
	DocumentFrequency map[string]int
	Hash              string
}

type chunkLinkCandidate struct {
	Index       int
	Score       float64
	SharedTerms []string
}

func buildUserChunkLinkCorpus(vault string) (chunkLinkCorpus, error) {
	inventory, err := buildNexusChunkInventory(vault)
	if err != nil {
		return chunkLinkCorpus{}, err
	}

	corpus := chunkLinkCorpus{Postings: map[string][]int{}, DocumentFrequency: map[string]int{}}
	ordinalByPath := map[string]int{}
	hash := sha256.New()
	for _, chunk := range inventory.Chunks {
		if chunk.Authorship != "nexus_user" {
			continue
		}
		ordinal := ordinalByPath[chunk.Path]
		ordinalByPath[chunk.Path]++
		terms := chunkLinkTerms(chunk.Text)
		idHash := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", chunk.Path, ordinal)))
		node := chunkLinkNode{
			ID:      "uch-" + hex.EncodeToString(idHash[:])[:20],
			Path:    chunk.Path,
			Ordinal: ordinal,
			Excerpt: connectionExcerpt(chunk.Text),
			Terms:   terms,
		}
		index := len(corpus.Nodes)
		corpus.Nodes = append(corpus.Nodes, node)
		corpus.Terms = append(corpus.Terms, terms)
		for _, term := range terms {
			corpus.Postings[term] = append(corpus.Postings[term], index)
			corpus.DocumentFrequency[term]++
		}
		fmt.Fprintf(hash, "%s\x00%s\x00%d\x00%s\n", node.ID, node.Path, node.Ordinal, chunk.Text)
	}
	corpus.Hash = hex.EncodeToString(hash.Sum(nil))
	return corpus, nil
}

func chunkLinkTerms(text string) []string {
	terms := uniqueStrings(connectionTerms(text))
	out := terms[:0]
	for _, term := range terms {
		if weaveLinkNoiseTerms[term] || connectionStopWords[term] || len(term) < 5 {
			continue
		}
		out = append(out, term)
	}
	sort.Strings(out)
	return out
}

func rankChunkLinkCandidates(corpus chunkLinkCorpus, source int, limit int) []chunkLinkCandidate {
	if source < 0 || source >= len(corpus.Nodes) || limit <= 0 {
		return nil
	}
	sharedByTarget := map[int]map[string]bool{}
	terms := corpus.Terms[source]
	for _, term := range terms {
		df := corpus.DocumentFrequency[term]
		// Very common words create a dense, meaningless graph. Keep the
		// index bounded and let rare terms carry discovery.
		if df > 800 {
			continue
		}
		for _, target := range corpus.Postings[term] {
			if target == source || corpus.Nodes[target].Path == corpus.Nodes[source].Path {
				continue
			}
			if sharedByTarget[target] == nil {
				sharedByTarget[target] = map[string]bool{}
			}
			sharedByTarget[target][term] = true
		}
	}

	out := make([]chunkLinkCandidate, 0, len(sharedByTarget))
	for target, sharedSet := range sharedByTarget {
		shared := make([]string, 0, len(sharedSet))
		score := 0.0
		for term := range sharedSet {
			shared = append(shared, term)
			df := corpus.DocumentFrequency[term]
			score += 1 / math.Log2(float64(df)+2)
		}
		sort.Strings(shared)
		if len(shared) < 2 {
			if len(shared) == 0 || corpus.DocumentFrequency[shared[0]] > 8 || len(shared[0]) < 8 {
				continue
			}
		}
		score += float64(len(shared)) * 0.5
		out = append(out, chunkLinkCandidate{Index: target, Score: score, SharedTerms: shared})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return corpus.Nodes[out[i].Index].ID < corpus.Nodes[out[j].Index].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func chunkLinkEdgeFor(corpus chunkLinkCorpus, source int, candidate chunkLinkCandidate) chunkLinkEdge {
	a, b := corpus.Nodes[source], corpus.Nodes[candidate.Index]
	h := sha256.Sum256([]byte(a.ID + "\x00" + b.ID))
	return chunkLinkEdge{
		ID:            "cle-" + hex.EncodeToString(h[:])[:20],
		SourceID:      a.ID,
		TargetID:      b.ID,
		SourcePath:    a.Path,
		TargetPath:    b.Path,
		SharedTerms:   candidate.SharedTerms,
		Score:         candidate.Score,
		Relationship:  "same_subject",
		Assessment:    "user-only chunk lexical discovery",
		Uncertainty:   "Candidate edge; shared language is evidence for review, not proof of authorial intent.",
		SourceExcerpt: a.Excerpt,
		TargetExcerpt: b.Excerpt,
	}
}

func loadChunkLinkGraph(path string) (chunkLinkGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return chunkLinkGraph{}, err
	}
	var graph chunkLinkGraph
	if err := json.Unmarshal(data, &graph); err != nil {
		return chunkLinkGraph{}, err
	}
	if graph.Version != chunkLinkGraphVersion {
		return chunkLinkGraph{}, fmt.Errorf("unsupported chunk graph version %q", graph.Version)
	}
	return graph, nil
}

func loadChunkLinkState(path string) (chunkLinkState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return chunkLinkState{}, err
	}
	var state chunkLinkState
	if err := json.Unmarshal(data, &state); err != nil {
		return chunkLinkState{}, err
	}
	if state.Version != chunkLinkGraphVersion || state.Status != "active" && state.Status != "completed" {
		return chunkLinkState{}, errors.New("invalid user chunk link state")
	}
	if state.NextOffset < 0 || state.NextOffset > state.TotalChunks {
		return chunkLinkState{}, errors.New("invalid user chunk link cursor")
	}
	return state, nil
}

func renderChunkLinkReport(graph chunkLinkGraph, state chunkLinkState) string {
	var b strings.Builder
	b.WriteString("# Pantheon user-only chunk link graph\n\n")
	b.WriteString("> Derived graph only. No Vault notes were modified. Nodes contain user-authored Nexus chunks; agent-authored chunks were excluded. Edges are lexical discovery candidates, not approved facts.\n\n")
	fmt.Fprintf(&b, "- Status: %s\n- Nodes: %d\n- Edges: %d\n- Processed: %d/%d chunks\n- Completed batches: %d\n- Inventory hash: `%s`\n\n", state.Status, graph.NodeCount, graph.EdgeCount, state.NextOffset, state.TotalChunks, state.CompletedBatches, state.InventoryHash)
	b.WriteString("## Strongest discovered edges\n\n")
	edges := append([]chunkLinkEdge(nil), graph.Edges...)
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].Score > edges[j].Score })
	if len(edges) > 200 {
		edges = edges[:200]
	}
	for _, edge := range edges {
		fmt.Fprintf(&b, "- `%s` — `%s` ↔ `%s` — shared: `%s`\n  - %s\n  - %s\n\n", edge.ID, edge.SourcePath, edge.TargetPath, strings.Join(edge.SharedTerms, ", "), edge.SourceExcerpt, edge.TargetExcerpt)
	}
	return b.String()
}

func runChunkLinkCrawler(statePath, graphPath, reportPath, vault string, batchSize, maxLinks int) error {
	if batchSize <= 0 || batchSize > 5000 {
		return errors.New("chunk crawl batch size must be between 1 and 5000")
	}
	if maxLinks <= 0 || maxLinks > 20 {
		return errors.New("maximum links per chunk must be between 1 and 20")
	}
	corpus, err := buildUserChunkLinkCorpus(vault)
	if err != nil {
		return err
	}
	if len(corpus.Nodes) == 0 {
		return errors.New("no user-authored Nexus chunks found")
	}

	state, err := loadChunkLinkState(statePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		state = chunkLinkState{Version: chunkLinkGraphVersion, RunID: now, CreatedAt: now, UpdatedAt: now, VaultPath: vault, InventoryHash: corpus.Hash, BatchSize: batchSize, MaxLinks: maxLinks, TotalChunks: len(corpus.Nodes), GraphPath: graphPath, Status: "active"}
		graph := chunkLinkGraph{Version: chunkLinkGraphVersion, GeneratedAt: now, VaultPath: vault, InventoryHash: corpus.Hash, Authorship: "nexus_user", NodeCount: len(corpus.Nodes), Nodes: corpus.Nodes}
		if err := writeJSONAtomic(graphPath, graph, 0o600); err != nil {
			return err
		}
		if err := writeJSONAtomic(statePath, state, 0o600); err != nil {
			return err
		}
	} else {
		if state.Status == "completed" {
			return fmt.Errorf("chunk crawl %s is already complete; use a new --state path for another crawl", state.RunID)
		}
		if state.VaultPath != vault || state.InventoryHash != corpus.Hash || state.TotalChunks != len(corpus.Nodes) {
			return errors.New("user chunk inventory changed; start a new chunk crawl state")
		}
		if state.BatchSize != batchSize || state.MaxLinks != maxLinks {
			return errors.New("existing chunk crawl uses different batch settings")
		}
	}

	graph, err := loadChunkLinkGraph(graphPath)
	if err != nil {
		return fmt.Errorf("load chunk graph: %w", err)
	}
	seen := map[string]bool{}
	for _, edge := range graph.Edges {
		seen[edgePairKey(edge.SourceID, edge.TargetID)] = true
	}
	for state.Status == "active" {
		end := min(state.NextOffset+state.BatchSize, len(corpus.Nodes))
		for source := state.NextOffset; source < end; source++ {
			for _, candidate := range rankChunkLinkCandidates(corpus, source, state.MaxLinks) {
				edge := chunkLinkEdgeFor(corpus, source, candidate)
				key := edgePairKey(edge.SourceID, edge.TargetID)
				if seen[key] {
					continue
				}
				seen[key] = true
				graph.Edges = append(graph.Edges, edge)
			}
		}
		state.NextOffset = end
		state.CompletedBatches++
		if state.NextOffset >= len(corpus.Nodes) {
			state.Status = "completed"
		}
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		graph.GeneratedAt = state.UpdatedAt
		graph.EdgeCount = len(graph.Edges)
		if err := writeJSONAtomic(graphPath, graph, 0o600); err != nil {
			return err
		}
		if err := writeBytesAtomic(reportPath, []byte(renderChunkLinkReport(graph, state)), 0o600); err != nil {
			return err
		}
		if err := writeJSONAtomic(statePath, state, 0o600); err != nil {
			return err
		}
		fmt.Printf("Chunk crawl batch %d: %d/%d chunks, %d edges\n", state.CompletedBatches, state.NextOffset, len(corpus.Nodes), len(graph.Edges))
	}
	fmt.Printf("Chunk crawl %s complete: %d user chunks, %d edges\n", state.RunID, len(corpus.Nodes), len(graph.Edges))
	return nil
}

func edgePairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}
