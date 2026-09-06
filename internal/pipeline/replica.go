package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/Thrapis/go-csv-translator/internal/extract"
	"github.com/Thrapis/go-csv-translator/internal/markup"
	"github.com/Thrapis/go-csv-translator/internal/textutil"
)

// translateRows translates each row independently.
func (p *Pipeline) translateRows(ctx context.Context, lines []extract.DataLine) error {
	for i := range lines {
		if err := ctx.Err(); err != nil {
			return err
		}
		translated, err := p.translateString(ctx, p.analyzer.Analyze(lines[i].Value))
		if err != nil {
			return err
		}
		p.log.Info("translated row",
			"progress", fmt.Sprintf("%d/%d", i+1, len(lines)),
			"from", truncate(lines[i].Value), "to", truncate(translated))
		lines[i].Value = translated
	}
	return nil
}

// translateReplicas groups consecutive rows that share a replica tag and
// translates each group as one unit. Rows with no tag (header sections) are
// each their own group. Groups touch disjoint slices of lines, so with
// Options.Concurrency > 1 they are translated by a bounded worker pool.
func (p *Pipeline) translateReplicas(ctx context.Context, lines []extract.DataLine) error {
	if p.grouper == nil {
		return p.translateRows(ctx, lines)
	}

	type span struct{ start, end int }
	var groups []span
	for i := 0; i < len(lines); {
		j := i + 1
		for j < len(lines) && p.grouper.SameReplica(lines[i], lines[j]) {
			j++
		}
		groups = append(groups, span{i, j - 1})
		i = j
	}

	conc := p.opts.Concurrency
	if conc < 1 {
		conc = 1
	}
	if conc == 1 {
		for _, g := range groups {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := p.translateReplicaGroup(ctx, lines, g.start, g.end); err != nil {
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
			if err := p.translateReplicaGroup(ctx, lines, g.start, g.end); err != nil {
				once.Do(func() { firstErr = err; cancel() })
			}
		}(g)
	}
	wg.Wait()
	return firstErr
}

func (p *Pipeline) translateReplicaGroup(ctx context.Context, lines []extract.DataLine, start, end int) error {
	var total string
	if p.opts.Parasitizing {
		for _, f := range p.opts.ParasitizingFiles {
			v, err := p.parasitizer.Replica(f, lines[start].Tag)
			if err != nil {
				return err
			}
			if strings.TrimSpace(v) != "" {
				total = v
				break
			}
		}
	}
	if strings.TrimSpace(total) == "" {
		// Not parasitizing, or no parasite match: translate the source rows.
		var b strings.Builder
		for j := start; j <= end; j++ {
			b.WriteString(lines[j].Value)
		}
		total = b.String()
	}
	if strings.TrimSpace(total) == "" {
		return nil
	}

	translated, err := p.translateString(ctx, p.analyzer.Analyze(total))
	if err != nil {
		return err
	}

	p.log.Info("translated replica",
		"rows", fmt.Sprintf("%d-%d", start, end),
		"from", truncate(lines[start].Value), "to", truncate(translated))

	if _, ok := p.analyzer.(markup.Masker); ok {
		// Whole translation on the first non-empty row; blank the rest.
		first := true
		for j := start; j <= end; j++ {
			if strings.TrimSpace(lines[j].Value) == "" {
				continue
			}
			if first {
				lines[j].Value, first = translated, false
			} else {
				lines[j].Value = ""
			}
		}
		return nil
	}

	// Fragment path: keep the row count, split the translation by word count.
	filled := 0
	for j := start; j <= end; j++ {
		if strings.TrimSpace(lines[j].Value) != "" {
			filled++
		}
	}
	parts := splitSentence(translated, filled)
	comp := 0
	for j := start; j <= end; j++ {
		if strings.TrimSpace(lines[j].Value) == "" {
			comp++
			continue
		}
		lines[j].Value = parts[j-start-comp]
	}
	return nil
}

// translateString translates one analyzed string and returns the rendered
// result. When the analyzer supports masking the whole string goes to the
// translator in one request with markup replaced by §i§ sentinels; otherwise
// each free-text fragment is translated on its own.
func (p *Pipeline) translateString(ctx context.Context, ps *markup.PartialString) (string, error) {
	masker, ok := p.analyzer.(markup.Masker)
	if !ok {
		if err := p.translateParts(ctx, ps); err != nil {
			return "", err
		}
		return p.analyzer.Render(ps), nil
	}

	masked, markers := masker.Mask(ps)
	bare := strings.TrimSpace(markup.StripSentinels(masked))
	if bare == "" {
		return p.analyzer.Render(ps), nil // markup only, nothing to translate
	}

	trimmed := strings.TrimSpace(masked)
	lead := textutil.CountLeadingSpaces(masked)
	trail := textutil.CountFinalSpaces(masked)

	out, err := p.translator.Translate(ctx, trimmed, p.opts.SourceLang, p.opts.TargetLang)
	if err != nil {
		return "", fmt.Errorf("translate %q: %w", truncate(trimmed), err)
	}

	if len(markers) > 0 && !markup.SentinelsIntact(out, len(markers)) {
		// The translator corrupted a sentinel; fall back to translating each
		// free-text fragment on its own (ps is still un-mutated here).
		if err := p.translateParts(ctx, ps); err != nil {
			return "", err
		}
		return p.analyzer.Render(ps), nil
	}

	out = markup.Unmask(out, markers)
	out = caseFirstLetter(out, firstLetterUpper(bare))
	return strings.Repeat(" ", lead) + out + strings.Repeat(" ", trail), nil
}

// translateParts translates every free-text part of ps in place, preserving each
// part's leading/trailing spaces and the case of its first letter. Used only for
// analyzers that do not implement markup.Masker.
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
