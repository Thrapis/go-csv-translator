package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/Thrapis/go-csv-translator/internal/extract"
	"github.com/Thrapis/go-csv-translator/internal/markup"
	"github.com/Thrapis/go-csv-translator/internal/textutil"
	"github.com/Thrapis/go-csv-translator/internal/translate"
)

// defaultBatchSize is used when Options.BatchSize is unset.
const defaultBatchSize = 64

// span is a half-open... actually inclusive [start, end] row range for one replica.
type span struct{ start, end int }

// translateRows translates each row independently. Masker analyzers go through a
// single batched pass; others fall back to one request per row.
func (p *Pipeline) translateRows(ctx context.Context, lines []extract.DataLine) error {
	if _, ok := p.analyzer.(markup.Masker); !ok {
		for i := range lines {
			if err := ctx.Err(); err != nil {
				return err
			}
			out, err := p.translateString(ctx, p.analyzer.Analyze(lines[i].Value))
			if err != nil {
				return err
			}
			lines[i].Value = out
		}
		return nil
	}

	sources := make([]string, len(lines))
	for i := range lines {
		sources[i] = lines[i].Value
	}
	outs, err := p.translateManyMasked(ctx, sources)
	if err != nil {
		return err
	}
	for i := range lines {
		if strings.TrimSpace(lines[i].Value) != "" {
			lines[i].Value = outs[i]
		}
	}
	return nil
}

// translateReplicas groups consecutive rows sharing a replica tag and translates
// each group as one unit (untagged rows are their own group). Masker analyzers
// use a batched pass; others use a bounded worker pool over the groups.
func (p *Pipeline) translateReplicas(ctx context.Context, lines []extract.DataLine) error {
	if p.grouper == nil {
		return p.translateRows(ctx, lines)
	}
	groups := p.replicaSpans(lines)

	if _, ok := p.analyzer.(markup.Masker); ok {
		return p.translateReplicasBatched(ctx, lines, groups)
	}
	return p.translateReplicasPool(ctx, lines, groups)
}

func (p *Pipeline) replicaSpans(lines []extract.DataLine) []span {
	var groups []span
	for i := 0; i < len(lines); {
		j := i + 1
		for j < len(lines) && p.grouper.SameReplica(lines[i], lines[j]) {
			j++
		}
		groups = append(groups, span{i, j - 1})
		i = j
	}
	return groups
}

// translateReplicasBatched prepares every group's source string, translates them
// all in batched requests, then writes each result back.
func (p *Pipeline) translateReplicasBatched(ctx context.Context, lines []extract.DataLine, groups []span) error {
	totals := make([]string, len(groups))
	fromParasite := 0
	for k, g := range groups {
		t, para, err := p.replicaTotal(lines, g)
		if err != nil {
			return err
		}
		totals[k] = t
		if para {
			fromParasite++
		}
	}
	if p.opts.Parasitizing {
		p.log.Info("replicas", "total", len(groups),
			"from_parasite", fromParasite, "machine_translated", len(groups)-fromParasite)
	}

	outs, err := p.translateManyMasked(ctx, totals)
	if err != nil {
		return err
	}

	for k, g := range groups {
		if strings.TrimSpace(totals[k]) == "" {
			continue
		}
		placeReplicaWhole(lines, g, outs[k])
	}
	return nil
}

// translateReplicasPool is the non-Masker path: groups touch disjoint slices, so
// they run through a bounded worker pool.
func (p *Pipeline) translateReplicasPool(ctx context.Context, lines []extract.DataLine, groups []span) error {
	conc := p.concurrency()
	if conc == 1 {
		for _, g := range groups {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := p.translateReplicaGroup(ctx, lines, g); err != nil {
				return err
			}
		}
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sem := make(chan struct{}, conc)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	for _, g := range groups {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(g span) {
			defer wg.Done()
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			if err := p.translateReplicaGroup(ctx, lines, g); err != nil {
				once.Do(func() { firstErr = err; cancel() })
			}
		}(g)
	}
	wg.Wait()
	return firstErr
}

func (p *Pipeline) translateReplicaGroup(ctx context.Context, lines []extract.DataLine, g span) error {
	total, _, err := p.replicaTotal(lines, g)
	if err != nil {
		return err
	}
	if strings.TrimSpace(total) == "" {
		return nil
	}

	translated, err := p.translateString(ctx, p.analyzer.Analyze(total))
	if err != nil {
		return err
	}

	// non-Masker: keep the row count, split the translation by word count.
	filled := 0
	for j := g.start; j <= g.end; j++ {
		if strings.TrimSpace(lines[j].Value) != "" {
			filled++
		}
	}
	parts := splitSentence(translated, filled)
	comp := 0
	for j := g.start; j <= g.end; j++ {
		if strings.TrimSpace(lines[j].Value) == "" {
			comp++
			continue
		}
		lines[j].Value = parts[j-g.start-comp]
	}
	return nil
}

// replicaTotal is the source text for a group: the first parasite file with a
// match (fromParasite true), else the group's own joined source rows.
func (p *Pipeline) replicaTotal(lines []extract.DataLine, g span) (text string, fromParasite bool, err error) {
	if p.opts.Parasitizing {
		for _, f := range p.opts.ParasitizingFiles {
			v, err := p.parasitizer.Replica(f, lines[g.start].Tag)
			if err != nil {
				return "", false, err
			}
			if strings.TrimSpace(v) != "" {
				return v, true, nil
			}
		}
	}
	var b strings.Builder
	for j := g.start; j <= g.end; j++ {
		b.WriteString(lines[j].Value)
	}
	return b.String(), false, nil
}

// placeReplicaWhole puts the whole translation on the first non-empty row of the
// group and blanks the rest.
func placeReplicaWhole(lines []extract.DataLine, g span, translated string) {
	first := true
	for j := g.start; j <= g.end; j++ {
		if strings.TrimSpace(lines[j].Value) == "" {
			continue
		}
		if first {
			lines[j].Value, first = translated, false
		} else {
			lines[j].Value = ""
		}
	}
}

// translateManyMasked runs the Masker whole-string pipeline over many source
// strings at once: mask each, batch-translate, unmask, restore case and outer
// spacing. A source that mangles its sentinels is retried through the
// fragment path. sources[i] that is empty or markup-only yields "" or the
// rendered original respectively.
func (p *Pipeline) translateManyMasked(ctx context.Context, sources []string) ([]string, error) {
	masker := p.analyzer.(markup.Masker)
	out := make([]string, len(sources))

	type meta struct {
		markers     []string
		bare        string
		lead, trail int
		src         string
	}
	m := make([]meta, len(sources))
	var idx []int
	var texts []string

	for i, s := range sources {
		if strings.TrimSpace(s) == "" {
			continue
		}
		ps := p.analyzer.Analyze(s)
		masked, markers := masker.Mask(ps)
		bare := strings.TrimSpace(markup.StripSentinels(masked))
		if bare == "" {
			out[i] = p.analyzer.Render(ps)
			continue
		}
		m[i] = meta{
			markers: markers, bare: bare,
			lead: textutil.CountLeadingSpaces(masked), trail: textutil.CountFinalSpaces(masked),
			src: s,
		}
		idx = append(idx, i)
		texts = append(texts, strings.TrimSpace(masked))
	}

	results, err := p.translateBatch(ctx, texts)
	if err != nil {
		return nil, err
	}

	for n, i := range idx {
		raw := results[n]
		md := m[i]
		if len(md.markers) > 0 && !markup.SentinelsIntact(raw, len(md.markers)) {
			ps := p.analyzer.Analyze(md.src)
			if err := p.translateParts(ctx, ps); err != nil {
				return nil, err
			}
			out[i] = p.analyzer.Render(ps)
			continue
		}
		u := markup.Unmask(raw, md.markers)
		u = caseFirstLetter(u, firstLetterUpper(md.bare))
		out[i] = strings.Repeat(" ", md.lead) + u + strings.Repeat(" ", md.trail)
	}
	return out, nil
}

// translateBatch translates texts in order, in chunks of batchSize, running up
// to Concurrency chunks at once. A chunk whose batched call fails or returns the
// wrong count is retried one string at a time.
func (p *Pipeline) translateBatch(ctx context.Context, texts []string) ([]string, error) {
	out := make([]string, len(texts))
	if len(texts) == 0 {
		return out, nil
	}

	size := p.opts.BatchSize
	if size < 1 {
		size = defaultBatchSize
	}
	var chunks []span
	for lo := 0; lo < len(texts); lo += size {
		hi := lo + size
		if hi > len(texts) {
			hi = len(texts)
		}
		chunks = append(chunks, span{lo, hi - 1})
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sem := make(chan struct{}, p.concurrency())
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error
	fail := func(err error) { once.Do(func() { firstErr = err; cancel() }) }

	total := len(texts)
	started := time.Now()
	var done int64

	for _, c := range chunks {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(lo, hi int) { // hi inclusive
			defer wg.Done()
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			in := texts[lo : hi+1]
			res, err := translate.Batch(ctx, p.translator, in, p.opts.SourceLang, p.opts.TargetLang)
			if err == nil && len(res) == len(in) {
				copy(out[lo:hi+1], res)
			} else {
				p.log.Warn("batch fell back to per-string", "reason", err)
				for i := lo; i <= hi; i++ {
					v, e := p.translator.Translate(ctx, texts[i], p.opts.SourceLang, p.opts.TargetLang)
					if e != nil {
						fail(fmt.Errorf("translate %q: %w", truncate(texts[i]), e))
						return
					}
					out[i] = v
				}
			}
			p.reportProgress(atomic.AddInt64(&done, int64(hi-lo+1)), int64(total), started)
		}(c.start, c.end)
	}
	wg.Wait()
	return out, firstErr
}

// reportProgress logs a single "translating" line per completed chunk.
func (p *Pipeline) reportProgress(done, total int64, started time.Time) {
	if total == 0 {
		return
	}
	elapsed := time.Since(started)
	var eta time.Duration
	if done > 0 {
		eta = (elapsed * time.Duration(total-done) / time.Duration(done)).Round(time.Second)
	}
	p.log.Info("translating",
		"done", fmt.Sprintf("%d/%d", done, total),
		"pct", done*100/total,
		"eta", eta)
}

func (p *Pipeline) concurrency() int {
	if p.opts.Concurrency < 1 {
		return 1
	}
	return p.opts.Concurrency
}

// translateString translates one analyzed string for the non-Masker path: each
// free-text fragment is translated on its own and reassembled.
func (p *Pipeline) translateString(ctx context.Context, ps *markup.PartialString) (string, error) {
	if err := p.translateParts(ctx, ps); err != nil {
		return "", err
	}
	return p.analyzer.Render(ps), nil
}

// translateParts translates every free-text part of ps in place, preserving each
// part's leading/trailing spaces and the case of its first letter.
func (p *Pipeline) translateParts(ctx context.Context, ps *markup.PartialString) error {
	for _, part := range p.analyzer.Translatable(ps) {
		trimmed := strings.TrimSpace(part.Value)
		if trimmed == "" {
			continue
		}
		lead := textutil.CountLeadingSpaces(part.Value)
		trail := textutil.CountFinalSpaces(part.Value)
		upperFirst := textutil.IsUpper([]rune(trimmed)[0])

		out, err := p.translator.Translate(ctx, trimmed, p.opts.SourceLang, p.opts.TargetLang)
		if err != nil {
			return fmt.Errorf("translate %q: %w", truncate(trimmed), err)
		}
		out = applyFirstCase(out, upperFirst)
		part.Value = strings.Repeat(" ", lead) + out + strings.Repeat(" ", trail)
	}
	return nil
}

func firstLetterUpper(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return unicode.IsUpper(r)
		}
	}
	return false
}

// caseFirstLetter adjusts the case of the first Unicode letter in s (skipping
// any leading markup), leaving everything else untouched.
func caseFirstLetter(s string, upper bool) string {
	for i, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		c := unicode.ToLower(r)
		if upper {
			c = unicode.ToUpper(r)
		}
		return s[:i] + string(c) + s[i+len(string(r)):]
	}
	return s
}

func applyFirstCase(s string, upper bool) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	head := string(r[0])
	if upper {
		head = strings.ToUpper(head)
	} else {
		head = strings.ToLower(head)
	}
	return head + string(r[1:])
}

// splitSentence distributes the words of sentence as evenly as possible into n
// parts. If there are fewer words than parts, the trailing parts are empty.
func splitSentence(sentence string, n int) []string {
	if n <= 0 {
		return []string{}
	}
	words := strings.Fields(sentence)
	total := len(words)

	if total < n {
		out := make([]string, n)
		copy(out, words)
		return out
	}

	base := total / n
	rem := total % n
	out := make([]string, 0, n)
	start := 0
	for i := 0; i < n; i++ {
		end := start + base
		if i < rem {
			end++
		}
		out = append(out, strings.Join(words[start:end], " "))
		start = end
	}
	return out
}

func truncate(s string) string {
	const max = 48
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
