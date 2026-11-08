package pantheon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	breadPromotionBegin = "<!-- PANTHEON-BREAD-PROMOTION:BEGIN -->"
	breadPromotionEnd   = "<!-- PANTHEON-BREAD-PROMOTION:END -->"
	breadPromotionNote  = "00 Pantheon Core/Curated Graph/Bread Standard/Bread Standard — Promoted User Shards.md"
	breadIndexNote      = "00 Pantheon Core/Curated Graph/Bread Standard/Bread Standard — Index.md"
)

type breadPromotionCandidate struct {
	Node        chunkLinkNode
	Score       float64
	SharedTerms []string
	SourceText  string
}

type breadPromotionJournal struct {
	Version       string                  `json:"version"`
	RunID         string                  `json:"run_id"`
	StartedAt     string                  `json:"started_at"`
	FinishedAt    string                  `json:"finished_at,omitempty"`
	VaultPath     string                  `json:"vault_path"`
	PromotionNote string                  `json:"promotion_note"`
	Selected      int                     `json:"selected"`
	Status        string                  `json:"status"`
	Entries       []weaveLinkJournalEntry `json:"entries"`
}

func breadStandardTopicTerms(vault string) (map[string]bool, error) {
	paths := []string{
		breadIndexNote,
		"00 Pantheon Core/Curated Graph/Bread Standard/Bread Standard — Omi Evidence Boundary.md",
		"00 Pantheon Core/Curated Graph/Bread Standard/Bread Standard — Socio Bridge.md",
		"00 Pantheon Core/Curated Graph/Bread Standard/Bread Standard — Supporting Frameworks.md",
		"00 Pantheon Core/Weave/Concepts/Bread Standard.md",
	}
	entries, err := os.ReadDir(filepath.Join(vault, "20 Projects", "Bread Standard"))
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				paths = append(paths, filepath.ToSlash(filepath.Join("20 Projects", "Bread Standard", entry.Name())))
			}
		}
	}
	terms := map[string]bool{}
	for _, rel := range paths {
		data, err := readRegularFileNoSymlink(filepath.Join(vault, filepath.FromSlash(rel)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, term := range connectionTerms(string(data)) {
			if !weaveLinkNoiseTerms[term] && !connectionStopWords[term] {
				terms[term] = true
			}
		}
	}
	terms["bread"] = true
	terms["standard"] = true
	return terms, nil
}

func selectBreadPromotionCandidates(corpus chunkLinkCorpus, fullText map[string]string, topicTerms map[string]bool, limit, maxPerSource int) []breadPromotionCandidate {
	candidates := make([]breadPromotionCandidate, 0)
	for i, node := range corpus.Nodes {
		shared := make([]string, 0)
		score := 0.0
		for _, term := range corpus.Terms[i] {
			if !topicTerms[term] {
				continue
			}
			shared = append(shared, term)
			df := corpus.DocumentFrequency[term]
			score += 1 / (1 + float64(df)/50)
		}
		text := fullText[node.ID]
		if strings.Contains(strings.ToLower(text), "bread standard") {
			score += 5
			shared = append(shared, "bread standard")
		}
		shared = uniqueStrings(shared)
		if len(shared) < 2 || score < 2 {
			continue
		}
		sort.Strings(shared)
		if len(shared) > 8 {
			shared = shared[:8]
		}
		candidates = append(candidates, breadPromotionCandidate{Node: node, Score: score, SharedTerms: shared, SourceText: text})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Node.ID < candidates[j].Node.ID
	})
	selected := make([]breadPromotionCandidate, 0, limit)
	perSource := map[string]int{}
	for _, candidate := range candidates {
		if perSource[candidate.Node.Path] >= maxPerSource {
			continue
		}
		selected = append(selected, candidate)
		perSource[candidate.Node.Path]++
		if len(selected) >= limit {
			break
		}
	}
	return selected
}

func renderBreadPromotionNote(selected []breadPromotionCandidate, shardRoot string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ncurated_graph: bread-standard\npromotion: user-shards\nstatus: review\nselected_shards: %d\n---\n\n# Bread Standard — Promoted User Shards\n\nThese user-authored shards were selected from the Nexus archive because their text overlaps the existing Bread Standard curated corpus. They are evidence promoted for human review, not canonical claims.\n\n", len(selected))
	for _, candidate := range selected {
		path := userShardRelativePath(shardRoot, candidate.Node.ID)
		fmt.Fprintf(&b, "- [[%s|%s]] — score `%.2f`; shared terms: `%s`; source: [[%s|%s]]\n", path, candidate.Node.ID, candidate.Score, strings.Join(candidate.SharedTerms, ", "), candidate.Node.Path, strings.TrimSuffix(filepath.Base(candidate.Node.Path), filepath.Ext(candidate.Node.Path)))
	}
	b.WriteString("\n> Promotion is bounded and reversible. The raw conversation archive remains preserved.\n")
	return []byte(b.String())
}

func renderBreadPromotionBlock(selected []breadPromotionCandidate, shardRoot string) string {
	var b strings.Builder
	b.WriteString(breadPromotionBegin + "\n\n## Promoted user evidence shards\n\n")
	fmt.Fprintf(&b, "- [[%s|Bread Standard — Promoted User Shards]] — human review surface for all %d selected shards\n", breadPromotionNote, len(selected))
	for _, candidate := range selected {
		path := userShardRelativePath(shardRoot, candidate.Node.ID)
		fmt.Fprintf(&b, "- [[%s|%s]] — score `%.2f`; shared terms: `%s`\n", path, candidate.Node.ID, candidate.Score, strings.Join(candidate.SharedTerms, ", "))
	}
	b.WriteString("\n> Generated by Pantheon from user-authored shard evidence. Review before treating these as canonical.\n\n")
	b.WriteString(breadPromotionEnd)
	return b.String()
}

func replaceOrAppendGeneratedBlock(content, begin, end, block string) ([]byte, error) {
	start, finish := strings.Index(content, begin), strings.Index(content, end)
	if start >= 0 || finish >= 0 {
		if start < 0 || finish < start {
			return nil, errors.New("malformed generated promotion block")
		}
		finish += len(end)
		return []byte(content[:start] + block + content[finish:]), nil
	}
	separator := "\n\n"
	if strings.HasSuffix(content, "\n") {
		separator = "\n"
	}
	return []byte(content + separator + block + "\n"), nil
}

func promoteBreadStandardShards(vault, stateDir, shardRoot string, limit, maxPerSource int) (string, error) {
	if limit <= 0 || limit > 500 || maxPerSource <= 0 || maxPerSource > 10 {
		return "", errors.New("promotion limits are out of range")
	}
	corpus, err := buildUserChunkLinkCorpus(vault)
	if err != nil {
		return "", err
	}
	inventory, err := buildNexusChunkInventory(vault)
	if err != nil {
		return "", err
	}
	fullText := map[string]string{}
	ordinals := map[string]int{}
	for _, chunk := range inventory.Chunks {
		if chunk.Authorship != "nexus_user" {
			continue
		}
		ordinal := ordinals[chunk.Path]
		ordinals[chunk.Path]++
		fullText[userChunkID(chunk.Path, ordinal)] = chunk.Text
	}
	topicTerms, err := breadStandardTopicTerms(vault)
	if err != nil {
		return "", err
	}
	selected := selectBreadPromotionCandidates(corpus, fullText, topicTerms, limit, maxPerSource)
	if len(selected) == 0 {
		return "", errors.New("no Bread Standard shard candidates met the promotion threshold")
	}

	now := time.Now().UTC()
	runID := now.Format("20060102T150405.000000000Z")
	journalPath := filepath.Join(stateDir, "bread-promotion-"+runID+".json")
	backupDir := filepath.Join(stateDir, "backups", "bread-promotion-"+runID)
	journal := breadPromotionJournal{Version: chunkLinkGraphVersion, RunID: runID, StartedAt: now.Format(time.RFC3339Nano), VaultPath: vault, PromotionNote: breadPromotionNote, Selected: len(selected), Status: "running"}
	if err := writeJSONAtomic(journalPath, journal, 0o600); err != nil {
		return "", err
	}
	writeGenerated := func(rel string, after []byte) error {
		path := filepath.Join(vault, filepath.FromSlash(rel))
		before, err := readRegularFileNoSymlink(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && bytes.Equal(before, after) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		entry := weaveLinkJournalEntry{Path: rel, AfterHash: hashBytes(after), Action: "created"}
		if before != nil {
			if err := os.MkdirAll(backupDir, 0o700); err != nil {
				return err
			}
			backup := filepath.Join(backupDir, hashBytes([]byte(rel))+".md")
			if err := writeBytesAtomic(backup, before, 0o600); err != nil {
				return err
			}
			entry = weaveLinkJournalEntry{Path: rel, BackupPath: backup, BeforeHash: hashBytes(before), AfterHash: hashBytes(after), Action: "updated"}
		}
		if err := writeBytesAtomic(path, after, 0o600); err != nil {
			return err
		}
		journal.Entries = append(journal.Entries, entry)
		return writeJSONAtomic(journalPath, journal, 0o600)
	}

	if err := writeGenerated(breadPromotionNote, renderBreadPromotionNote(selected, shardRoot)); err != nil {
		return journalPath, err
	}
	indexBefore, err := readRegularFileNoSymlink(filepath.Join(vault, filepath.FromSlash(breadIndexNote)))
	if err != nil {
		return journalPath, err
	}
	indexAfter, err := replaceOrAppendGeneratedBlock(string(indexBefore), breadPromotionBegin, breadPromotionEnd, renderBreadPromotionBlock(selected, shardRoot))
	if err != nil {
		return journalPath, err
	}
	if err := writeGenerated(breadIndexNote, indexAfter); err != nil {
		return journalPath, err
	}
	for _, candidate := range selected {
		rel := userShardRelativePath(shardRoot, candidate.Node.ID)
		path := filepath.Join(vault, filepath.FromSlash(rel))
		before, err := readRegularFileNoSymlink(path)
		if err != nil {
			return journalPath, err
		}
		block := fmt.Sprintf("%s\n\n## Bread Standard promotion\n\n- [[%s|Bread Standard — Index]] — selected as user-authored evidence; score `%.2f`; shared terms: `%s`\n\n> This is a review promotion, not a canonical claim.\n\n%s", breadPromotionBegin, breadIndexNote, candidate.Score, strings.Join(candidate.SharedTerms, ", "), breadPromotionEnd)
		after, err := replaceOrAppendGeneratedBlock(string(before), breadPromotionBegin, breadPromotionEnd, block)
		if err != nil {
			return journalPath, err
		}
		if err := writeGenerated(rel, after); err != nil {
			return journalPath, err
		}
	}
	journal.Status = "completed"
	journal.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeJSONAtomic(journalPath, journal, 0o600); err != nil {
		return journalPath, err
	}
	return journalPath, nil
}

func promotionID(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])[:16]
}
