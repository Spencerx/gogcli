package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"google.golang.org/api/docs/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type DocsWriteCmd struct {
	DocID    string `arg:"" name:"docId" help:"Doc ID"`
	Text     string `name:"text" help:"Text to write"`
	File     string `name:"file" help:"Text file path ('-' for stdin)"`
	Append   bool   `name:"append" help:"Append instead of replacing the document body"`
	Pageless bool   `name:"pageless" help:"Set document to pageless mode"`
	TabID    string `name:"tab-id" help:"Target a specific tab by ID (see docs list-tabs)"`
}

func (c *DocsWriteCmd) Run(ctx context.Context, kctx *kong.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}

	text, provided, err := resolveTextInput(c.Text, c.File, kctx, "text", "file")
	if err != nil {
		return err
	}
	if !provided {
		return usage("required: --text or --file")
	}
	if text == "" {
		return usage("empty text")
	}

	svc, err := requireDocsService(ctx, flags)
	if err != nil {
		return err
	}

	endIndex, err := docsTargetEndIndex(ctx, svc, id, c.TabID)
	if err != nil {
		return err
	}
	insertIndex := int64(1)
	if c.Append {
		insertIndex = docsAppendIndex(endIndex)
	}

	var reqs []*docs.Request
	if !c.Append {
		deleteEnd := endIndex - 1
		if deleteEnd > 1 {
			reqs = append(reqs, &docs.Request{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{StartIndex: 1, EndIndex: deleteEnd, TabId: c.TabID},
				},
			})
		}
	}
	reqs = append(reqs, &docs.Request{
		InsertText: &docs.InsertTextRequest{
			Location: &docs.Location{Index: insertIndex, TabId: c.TabID},
			Text:     text,
		},
	})

	resp, err := svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{Requests: reqs}).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
		}
		return err
	}
	if c.Pageless {
		if err := setDocumentPageless(ctx, svc, id); err != nil {
			return fmt.Errorf("set pageless mode: %w", err)
		}
	}

	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"documentId": resp.DocumentId,
			"requests":   len(reqs),
			"append":     c.Append,
			"index":      insertIndex,
		}
		if c.TabID != "" {
			payload["tabId"] = c.TabID
		}
		if resp.WriteControl != nil {
			payload["writeControl"] = resp.WriteControl
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	u.Out().Printf("id\t%s", resp.DocumentId)
	u.Out().Printf("requests\t%d", len(reqs))
	u.Out().Printf("append\t%t", c.Append)
	u.Out().Printf("index\t%d", insertIndex)
	if c.TabID != "" {
		u.Out().Printf("tabId\t%s", c.TabID)
	}
	if resp.WriteControl != nil && resp.WriteControl.RequiredRevisionId != "" {
		u.Out().Printf("revision\t%s", resp.WriteControl.RequiredRevisionId)
	}
	return nil
}

type DocsUpdateCmd struct {
	DocID    string `arg:"" name:"docId" help:"Doc ID"`
	Text     string `name:"text" help:"Text to insert"`
	File     string `name:"file" help:"Text file path ('-' for stdin)"`
	Index    int64  `name:"index" help:"Insert index (default: end of document)"`
	Pageless bool   `name:"pageless" help:"Set document to pageless mode"`
	TabID    string `name:"tab-id" help:"Target a specific tab by ID (see docs list-tabs)"`
}

func (c *DocsUpdateCmd) Run(ctx context.Context, kctx *kong.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}

	text, provided, err := resolveTextInput(c.Text, c.File, kctx, "text", "file")
	if err != nil {
		return err
	}
	if !provided {
		return usage("required: --text or --file")
	}
	if text == "" {
		return usage("empty text")
	}
	if flagProvided(kctx, "index") && c.Index <= 0 {
		return usage("invalid --index (must be >= 1)")
	}

	svc, err := requireDocsService(ctx, flags)
	if err != nil {
		return err
	}

	insertIndex := c.Index
	if insertIndex <= 0 {
		endIndex, endErr := docsTargetEndIndex(ctx, svc, id, c.TabID)
		if endErr != nil {
			return endErr
		}
		insertIndex = docsAppendIndex(endIndex)
	}

	reqs := []*docs.Request{{
		InsertText: &docs.InsertTextRequest{
			Location: &docs.Location{Index: insertIndex, TabId: c.TabID},
			Text:     text,
		},
	}}

	resp, err := svc.Documents.BatchUpdate(id, &docs.BatchUpdateDocumentRequest{Requests: reqs}).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
		}
		return err
	}
	if c.Pageless {
		if err := setDocumentPageless(ctx, svc, id); err != nil {
			return fmt.Errorf("set pageless mode: %w", err)
		}
	}

	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"documentId": resp.DocumentId,
			"requests":   len(reqs),
			"index":      insertIndex,
		}
		if c.TabID != "" {
			payload["tabId"] = c.TabID
		}
		if resp.WriteControl != nil {
			payload["writeControl"] = resp.WriteControl
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	u.Out().Printf("id\t%s", resp.DocumentId)
	u.Out().Printf("requests\t%d", len(reqs))
	u.Out().Printf("index\t%d", insertIndex)
	if c.TabID != "" {
		u.Out().Printf("tabId\t%s", c.TabID)
	}
	if resp.WriteControl != nil && resp.WriteControl.RequiredRevisionId != "" {
		u.Out().Printf("revision\t%s", resp.WriteControl.RequiredRevisionId)
	}
	return nil
}

type DocsInsertCmd struct {
	DocID   string `arg:"" name:"docId" help:"Doc ID"`
	Content string `arg:"" optional:"" name:"content" help:"Text to insert (or use --file / stdin)"`
	Index   int64  `name:"index" help:"Character index to insert at (1 = beginning)" default:"1"`
	File    string `name:"file" short:"f" help:"Read content from file (use - for stdin)"`
	TabID   string `name:"tab-id" help:"Target a specific tab by ID (see docs list-tabs)"`
}

func (c *DocsInsertCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return usage("empty docId")
	}
	content, err := resolveContentInput(c.Content, c.File)
	if err != nil {
		return err
	}
	if content == "" {
		return usage("no content provided (use argument, --file, or stdin)")
	}
	if c.Index < 1 {
		return usage("--index must be >= 1 (index 0 is reserved)")
	}

	svc, err := requireDocsService(ctx, flags)
	if err != nil {
		return err
	}

	result, err := svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{{
			InsertText: &docs.InsertTextRequest{
				Text: content,
				Location: &docs.Location{
					Index: c.Index,
					TabId: c.TabID,
				},
			},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("inserting text: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		payload := map[string]any{"documentId": result.DocumentId, "inserted": len(content), "atIndex": c.Index}
		if c.TabID != "" {
			payload["tabId"] = c.TabID
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	u.Out().Printf("documentId\t%s", result.DocumentId)
	u.Out().Printf("inserted\t%d bytes", len(content))
	u.Out().Printf("atIndex\t%d", c.Index)
	if c.TabID != "" {
		u.Out().Printf("tabId\t%s", c.TabID)
	}
	return nil
}

type DocsDeleteCmd struct {
	DocID string `arg:"" name:"docId" help:"Doc ID"`
	Start int64  `name:"start" required:"" help:"Start index (>= 1)"`
	End   int64  `name:"end" required:"" help:"End index (> start)"`
	TabID string `name:"tab-id" help:"Target a specific tab by ID (see docs list-tabs)"`
}

func (c *DocsDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return usage("empty docId")
	}
	if c.Start < 1 {
		return usage("--start must be >= 1")
	}
	if c.End <= c.Start {
		return usage("--end must be greater than --start")
	}

	svc, err := requireDocsService(ctx, flags)
	if err != nil {
		return err
	}

	result, err := svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: &docs.Range{StartIndex: c.Start, EndIndex: c.End, TabId: c.TabID},
			},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("deleting content: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"documentId": result.DocumentId,
			"deleted":    c.End - c.Start,
			"startIndex": c.Start,
			"endIndex":   c.End,
		}
		if c.TabID != "" {
			payload["tabId"] = c.TabID
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	u.Out().Printf("documentId\t%s", result.DocumentId)
	u.Out().Printf("deleted\t%d characters", c.End-c.Start)
	u.Out().Printf("range\t%d-%d", c.Start, c.End)
	if c.TabID != "" {
		u.Out().Printf("tabId\t%s", c.TabID)
	}
	return nil
}

type DocsClearCmd struct {
	DocID string `arg:"" name:"docId" help:"Doc ID"`
}

func (c *DocsClearCmd) Run(ctx context.Context, flags *RootFlags) error {
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return usage("empty docId")
	}
	return (&DocsSedCmd{DocID: docID, Expression: `s/^$//`}).Run(ctx, flags)
}

type DocsFindReplaceCmd struct {
	DocID       string `arg:"" name:"docId" help:"Doc ID"`
	Find        string `arg:"" name:"find" help:"Text to find"`
	ReplaceText string `arg:"" optional:"" name:"replace" help:"Replacement text (omit when using --content-file)"`
	ContentFile string `name:"content-file" help:"Read replacement from a file instead of the positional argument."`
	MatchCase   bool   `name:"match-case" help:"Case-sensitive matching"`
	Format      string `name:"format" help:"Replacement format: plain|markdown. Markdown converts formatting, tables, and inline images; local images must be under --content-file's directory (or use remote URLs)." default:"plain" enum:"plain,markdown"`
	First       bool   `name:"first" help:"Replace only the first occurrence instead of all."`
	TabID       string `name:"tab-id" help:"Target a specific tab by ID (see docs list-tabs)"`
}

type DocsEditCmd struct {
	DocID      string `arg:"" name:"docId" help:"Doc ID"`
	Find       string `arg:"" name:"find" help:"Text to find"`
	ReplaceStr string `arg:"" name:"replace" help:"Replacement text"`
	MatchCase  bool   `name:"match-case" help:"Case-sensitive matching"`
}

func (c *DocsEditCmd) Run(ctx context.Context, flags *RootFlags) error {
	return (&DocsFindReplaceCmd{
		DocID:       c.DocID,
		Find:        c.Find,
		ReplaceText: c.ReplaceStr,
		MatchCase:   c.MatchCase,
	}).Run(ctx, flags)
}

func (c *DocsFindReplaceCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return usage("empty docId")
	}
	if c.Find == "" {
		return usage("find text cannot be empty")
	}

	replaceText, err := c.resolveReplaceText()
	if err != nil {
		return err
	}

	format := strings.ToLower(strings.TrimSpace(c.Format))
	if format == "" {
		format = docsContentFormatPlain
	}
	if c.TabID != "" && format == docsContentFormatMarkdown {
		return usage("--tab-id is not yet supported with --format markdown")
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	svc, err := requireDocsService(ctx, flags)
	if err != nil {
		return err
	}

	if !c.First && format == docsContentFormatPlain {
		return c.runReplaceAll(ctx, u, svc, docID, replaceText)
	}

	doc, targetDoc, err := c.loadTargetDocument(ctx, svc, docID)
	if err != nil {
		return err
	}
	if targetDoc == nil {
		return errors.New("doc not found")
	}

	if c.First {
		startIdx, endIdx, total := findTextInDoc(targetDoc, c.Find, c.MatchCase)
		if total == 0 {
			return c.printFirstResult(ctx, u, docID, replaceText, 0, 0)
		}
		if format == docsContentFormatMarkdown {
			err = c.runMarkdown(ctx, svc, account, doc, startIdx, endIdx, replaceText)
		} else {
			err = c.runPlain(ctx, svc, doc, startIdx, endIdx, replaceText)
		}
		if err != nil {
			return err
		}
		return c.printFirstResult(ctx, u, docID, replaceText, 1, total)
	}

	matches := findTextMatches(targetDoc, c.Find, c.MatchCase)
	for i := len(matches) - 1; i >= 0; i-- {
		if err = c.runMarkdown(ctx, svc, account, doc, matches[i].startIndex, matches[i].endIndex, replaceText); err != nil {
			return err
		}
		if i == 0 {
			continue
		}
		doc, _, err = c.loadTargetDocument(ctx, svc, docID)
		if err != nil {
			return fmt.Errorf("re-reading document: %w", err)
		}
	}

	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"documentId":   docID,
			"find":         c.Find,
			"replace":      replaceText,
			"replacements": len(matches),
		}
		if c.TabID != "" {
			payload["tabId"] = c.TabID
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	u.Out().Printf("documentId\t%s", docID)
	u.Out().Printf("find\t%s", c.Find)
	u.Out().Printf("replace\t%s", replaceText)
	u.Out().Printf("replacements\t%d", len(matches))
	if c.TabID != "" {
		u.Out().Printf("tabId\t%s", c.TabID)
	}
	return nil
}

const (
	docsContentFormatPlain    = "plain"
	docsContentFormatMarkdown = "markdown"
)

func (c *DocsFindReplaceCmd) runReplaceAll(ctx context.Context, u *ui.UI, svc *docs.Service, docID, replaceText string) error {
	req := &docs.ReplaceAllTextRequest{
		ContainsText: &docs.SubstringMatchCriteria{Text: c.Find, MatchCase: c.MatchCase},
		ReplaceText:  replaceText,
	}
	if c.TabID != "" {
		req.TabsCriteria = &docs.TabsCriteria{TabIds: []string{c.TabID}}
	}

	result, err := svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{{ReplaceAllText: req}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("find-replace: %w", err)
	}

	replacements := int64(0)
	if len(result.Replies) > 0 && result.Replies[0].ReplaceAllText != nil {
		replacements = result.Replies[0].ReplaceAllText.OccurrencesChanged
	}

	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"documentId":   result.DocumentId,
			"find":         c.Find,
			"replace":      replaceText,
			"replacements": replacements,
		}
		if c.TabID != "" {
			payload["tabId"] = c.TabID
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	u.Out().Printf("documentId\t%s", result.DocumentId)
	u.Out().Printf("find\t%s", c.Find)
	u.Out().Printf("replace\t%s", replaceText)
	u.Out().Printf("replacements\t%d", replacements)
	if c.TabID != "" {
		u.Out().Printf("tabId\t%s", c.TabID)
	}
	return nil
}

func (c *DocsFindReplaceCmd) loadTargetDocument(ctx context.Context, svc *docs.Service, docID string) (*docs.Document, *docs.Document, error) {
	getCall := svc.Documents.Get(docID).Context(ctx)
	if c.TabID != "" {
		getCall = getCall.IncludeTabsContent(true)
	}

	doc, err := getCall.Do()
	if err != nil {
		if isDocsNotFound(err) {
			return nil, nil, fmt.Errorf("doc not found or not a Google Doc (id=%s)", docID)
		}
		return nil, nil, err
	}
	if doc == nil {
		return nil, nil, errors.New("doc not found")
	}
	if c.TabID == "" {
		return doc, doc, nil
	}

	tab := findTabByID(flattenTabs(doc.Tabs), c.TabID)
	if tab == nil {
		return nil, nil, fmt.Errorf("tab not found: %s", c.TabID)
	}
	if tab.DocumentTab == nil || tab.DocumentTab.Body == nil {
		return nil, nil, fmt.Errorf("tab has no document body: %s", c.TabID)
	}
	return doc, &docs.Document{
		DocumentId: doc.DocumentId,
		RevisionId: doc.RevisionId,
		Body:       tab.DocumentTab.Body,
	}, nil
}

func (c *DocsFindReplaceCmd) runPlain(ctx context.Context, svc *docs.Service, doc *docs.Document, startIdx, endIdx int64, replaceText string) error {
	_, err := svc.Documents.BatchUpdate(doc.DocumentId, &docs.BatchUpdateDocumentRequest{
		WriteControl: &docs.WriteControl{RequiredRevisionId: doc.RevisionId},
		Requests: []*docs.Request{
			{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{StartIndex: startIdx, EndIndex: endIdx, TabId: c.TabID},
				},
			},
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: startIdx, TabId: c.TabID},
					Text:     replaceText,
				},
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("replace: %w", err)
	}
	return nil
}

func (c *DocsFindReplaceCmd) runMarkdown(ctx context.Context, svc *docs.Service, account string, doc *docs.Document, startIdx, endIdx int64, replaceText string) error {
	cleaned, images := extractMarkdownImages(replaceText)
	elements := ParseMarkdown(cleaned)
	formattingRequests, textToInsert, tables := MarkdownToDocsRequests(elements, startIdx)

	requests := make([]*docs.Request, 0, 2+len(formattingRequests))
	requests = append(requests,
		&docs.Request{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: &docs.Range{StartIndex: startIdx, EndIndex: endIdx},
			},
		},
		&docs.Request{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: startIdx},
				Text:     textToInsert,
			},
		},
	)
	requests = append(requests, formattingRequests...)

	_, err := svc.Documents.BatchUpdate(doc.DocumentId, &docs.BatchUpdateDocumentRequest{
		WriteControl: &docs.WriteControl{RequiredRevisionId: doc.RevisionId},
		Requests:     requests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("replace (markdown): %w", err)
	}

	if len(tables) > 0 {
		tableInserter := NewTableInserter(svc, doc.DocumentId)
		tableOffset := int64(0)
		for _, table := range tables {
			tableIndex := table.StartIndex + tableOffset
			tableEnd, tableErr := tableInserter.InsertNativeTable(ctx, tableIndex, table.Cells)
			if tableErr != nil {
				return fmt.Errorf("insert native table: %w", tableErr)
			}
			if tableEnd > tableIndex {
				tableOffset += (tableEnd - tableIndex) - 1
			}
		}
	}

	if len(images) > 0 {
		imgErr := c.insertImages(ctx, account, svc, doc.DocumentId, images)
		cleanupImagePlaceholders(ctx, svc, doc.DocumentId, images)
		if imgErr != nil {
			return fmt.Errorf("insert images: %w", imgErr)
		}
	}

	return nil
}

func (c *DocsFindReplaceCmd) insertImages(ctx context.Context, account string, svc *docs.Service, docID string, images []markdownImage) error {
	basePath := c.ContentFile
	if basePath == "" {
		basePath = "."
	}
	return insertImagesIntoDocs(ctx, account, svc, docID, images, basePath)
}

func (c *DocsFindReplaceCmd) printFirstResult(ctx context.Context, u *ui.UI, docID, replaceText string, replacements, total int) error {
	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"documentId":   docID,
			"find":         c.Find,
			"replacements": replacements,
			"remaining":    total - replacements,
		}
		if c.TabID != "" {
			payload["tabId"] = c.TabID
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}

	u.Out().Printf("documentId\t%s", docID)
	u.Out().Printf("find\t%s", c.Find)
	u.Out().Printf("replace\t%s", replaceText)
	u.Out().Printf("replacements\t%d", replacements)
	if remaining := total - replacements; remaining > 0 {
		u.Out().Printf("remaining\t%d", remaining)
	}
	if c.TabID != "" {
		u.Out().Printf("tabId\t%s", c.TabID)
	}
	return nil
}

func (c *DocsFindReplaceCmd) resolveReplaceText() (string, error) {
	if c.ContentFile != "" && c.ReplaceText != "" {
		return "", usage("cannot use both replace argument and --content-file")
	}
	if c.ContentFile == "" {
		return c.ReplaceText, nil
	}
	data, err := os.ReadFile(c.ContentFile)
	if err != nil {
		return "", fmt.Errorf("read content file: %w", err)
	}
	return string(data), nil
}

func cleanupImagePlaceholders(ctx context.Context, svc *docs.Service, docID string, images []markdownImage) {
	reqs := make([]*docs.Request, 0, len(images))
	for _, img := range images {
		reqs = append(reqs, &docs.Request{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{
					Text:      img.placeholder(),
					MatchCase: true,
				},
				ReplaceText: "",
			},
		})
	}
	_, _ = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: reqs,
	}).Context(ctx).Do()
}

func findTextInDoc(doc *docs.Document, searchText string, matchCase bool) (int64, int64, int) {
	matches := findTextMatches(doc, searchText, matchCase)
	if len(matches) == 0 {
		return 0, 0, 0
	}
	return matches[0].startIndex, matches[0].endIndex, len(matches)
}

func findTextMatches(doc *docs.Document, searchText string, matchCase bool) []docRange {
	if doc == nil || doc.Body == nil {
		return nil
	}

	find := searchText
	if !matchCase {
		find = strings.ToLower(find)
	}

	var matches []docRange
	findTextInElements(doc.Body.Content, searchText, find, matchCase, &matches)
	return matches
}

func findTextInElements(elements []*docs.StructuralElement, searchText, find string, matchCase bool, matches *[]docRange) {
	for _, el := range elements {
		if el == nil {
			continue
		}
		switch {
		case el.Paragraph != nil:
			findTextInParagraph(el.Paragraph, searchText, find, matchCase, matches)
		case el.Table != nil:
			for _, row := range el.Table.TableRows {
				for _, cell := range row.TableCells {
					findTextInElements(cell.Content, searchText, find, matchCase, matches)
				}
			}
		}
	}
}

func findTextInParagraph(para *docs.Paragraph, searchText, find string, matchCase bool, matches *[]docRange) {
	var paraText strings.Builder
	var paraStart int64
	first := true
	for _, pe := range para.Elements {
		if pe.TextRun == nil {
			continue
		}
		if first {
			paraStart = pe.StartIndex
			first = false
		}
		paraText.WriteString(pe.TextRun.Content)
	}
	if paraText.Len() == 0 {
		return
	}

	text := paraText.String()
	compareText := text
	if !matchCase {
		compareText = strings.ToLower(text)
	}

	offset := 0
	for {
		idx := strings.Index(compareText[offset:], find)
		if idx < 0 {
			break
		}
		absIdx := offset + idx
		matchStart := paraStart + utf16Len(text[:absIdx])
		matchEnd := matchStart + utf16Len(searchText)
		*matches = append(*matches, docRange{startIndex: matchStart, endIndex: matchEnd})
		offset = absIdx + len(find)
	}
}
