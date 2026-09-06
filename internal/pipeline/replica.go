package pipeline

import (
	"context"
	"fmt"
	"strings"

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
		ps := p.analyzer.Analyze(lines[i].Value)
		if err := p.translateParts(ctx, ps); err != nil {
			return err
		}
		translated := p.analyzer.Render(ps)
		p.log.Info("translated row",
			"progress", fmt.Sprintf("%d/%d", i+1, len(lines)),
			"from", truncate(lines[i].Value), "to", truncate(translated))
		lines[i].Value = translated
	}
	return nil
}

// translateReplicas groups consecutive rows that share a replica tag, translates
// each group as one sentence, then redistributes the result across the group's
// non-empty rows.
//
// BUG(pre-existing): the loop control mirrors the original implementation, which
// never processes the final replica group in a file. Preserved deliberately so
// this refactor does not change output; fix separately.
func (p *Pipeline) translateReplicas(ctx context.Context, lines []extract.DataLine) error {
	if p.grouper == nil {
		return p.translateRows(ctx, lines)
	}

	start := 0
	for i := range lines {
		if start == i && i != len(lines)-1 {
			continue
		}
		if start != i && p.grouper.SameReplica(lines[start], lines[i]) {
			continue
		}
		end := i - 1
		if err := p.translateReplicaGroup(ctx, lines, start, end); err != nil {
			return err
		}
		start = end + 1
	}
	return nil
}

func (p *Pipeline) translateReplicaGroup(ctx context.Context, lines []extract.DataLine, start, end int) error {
	var total string
	if p.opts.Parasitizing {
		v, err := p.parasitizer.Replica(p.opts.ParasitizingFile, lines[start].Tag)
		if err != nil {
			return err
		}
		total = v // may be "" when the replica id is not in the parasite file
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

	filled := 0
	for j := start; j <= end; j++ {
		if strings.TrimSpace(lines[j].Value) != "" {
			filled++
		}
	}

	ps := p.analyzer.Analyze(total)
	if err := p.translateParts(ctx, ps); err != nil {
		return err
	}
	translated := p.analyzer.Render(ps)

	p.log.Info("translated replica",
		"rows", fmt.Sprintf("%d-%d", start, end),
		"from", truncate(lines[start].Value), "to", truncate(translated))

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
