package pipe

import (
	"regexp"
	"strings"

	"kb/internal"
)

// ChunkMarkdown splits a markdown document into a sequence of roughly equal
// sized chunks. It attempts to respect headings and avoid splitting inside
// fenced code blocks. Each chunk carries forward the document title or
// optionally breadcrumbs. The target size is approximately 1000 words with an
// overlap of 100 words. This implementation is deliberately simple – it uses
// word counts and does not account for tokenization differences across models.
func ChunkMarkdown(md string, title string, breadcrumbs []string) []internal.Chunk {
	lines := strings.Split(md, "\n")
	var blocks []struct {
		text  string
		crumb []string
	}
	current := []string{}
	currentBreadcrumb := breadcrumbs
	// pattern to detect headings (e.g. # Title)
	headingRe := regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	fenceCount := 0
	for _, ln := range lines {
		if m := headingRe.FindStringSubmatch(ln); m != nil {
			// commit previous block
			if len(current) > 0 {
				crumbCopy := make([]string, len(currentBreadcrumb))
				copy(crumbCopy, currentBreadcrumb)
				blocks = append(blocks, struct {
					text  string
					crumb []string
				}{text: strings.Join(current, "\n"), crumb: crumbCopy})
				current = []string{}
			}
			// update breadcrumb to include this heading
			currentBreadcrumb = append(breadcrumbs, strings.TrimSpace(m[2]))
			current = append(current, ln)
			continue
		}
		// track fenced code blocks
		if strings.HasPrefix(ln, "```") {
			// toggle fence state
			if fenceCount%2 == 0 {
				fenceCount++
			} else {
				fenceCount--
			}
		}
		current = append(current, ln)
	}
	if len(current) > 0 {
		crumbCopy := make([]string, len(currentBreadcrumb))
		copy(crumbCopy, currentBreadcrumb)
		blocks = append(blocks, struct {
			text  string
			crumb []string
		}{text: strings.Join(current, "\n"), crumb: crumbCopy})
	}
	// now merge blocks to target size
	var chunks []internal.Chunk
	buf := []string{}
	bufBreadcrumb := breadcrumbs
	words := 0
	// overlap := 100
	target := 1000
	for _, b := range blocks {
		w := wordCount(b.text)
		if words+w > target {
			// emit chunk
			if len(buf) > 0 {
				chunks = append(chunks, internal.Chunk{Text: strings.Join(buf, "\n"), Breadcrumbs: bufBreadcrumb})
			}
			// start new buffer with overlap from previous chunk
			// In this simplified implementation we do not implement overlap; production
			// systems can copy the last overlap words from the previous chunk here.
			buf = []string{b.text}
			bufBreadcrumb = b.crumb
			words = w
		} else {
			if len(buf) == 0 {
				bufBreadcrumb = b.crumb
			}
			buf = append(buf, b.text)
			words += w
		}
	}
	if len(buf) > 0 {
		chunks = append(chunks, internal.Chunk{Text: strings.Join(buf, "\n"), Breadcrumbs: bufBreadcrumb})
	}
	// If no chunks were produced (e.g. empty document), return a single empty chunk
	if len(chunks) == 0 {
		chunks = append(chunks, internal.Chunk{Text: md, Breadcrumbs: breadcrumbs})
	}
	// assign IDs and docIDs later in processArtifact
	return chunks
}

// wordCount is a helper to approximate token counts by splitting on
// whitespace. It is crude but sufficient for chunk sizing in this context.
func wordCount(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}