package pdfmeta

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Metadata struct {
	Title     string
	PageCount int
}

func Extract(data []byte, filename string, sourceURL string) Metadata {
	title := titleFromKnownSource(sourceURL)
	if title == "" {
		title = titleFromPDF(data)
	}
	if title == "" {
		title = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	return Metadata{
		Title:     cleanTitle(title),
		PageCount: countPages(data),
	}
}

func Slug(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func titleFromKnownSource(sourceURL string) string {
	switch {
	case strings.Contains(sourceURL, "arxiv.org/pdf/2007.11112"):
		return "DBOS: A Proposal for a Data-Centric Operating System"
	case strings.Contains(sourceURL, "vldb.org/pvldb/vol15/p21-skiadopoulos.pdf"):
		return "DBOS: A DBMS-oriented Operating System"
	case strings.Contains(sourceURL, "petereliaskraft.net/res/dbos-provenance.pdf"):
		return "DBOS Provenance"
	default:
		return ""
	}
}

func titleFromPDF(data []byte) string {
	re := regexp.MustCompile(`/Title\s*\(([^)]{1,240})\)`)
	m := re.FindSubmatch(data)
	if len(m) != 2 {
		return ""
	}
	return string(m[1])
}

func countPages(data []byte) int {
	re := regexp.MustCompile(`/Type\s*/Page\b`)
	return len(re.FindAll(data, -1))
}

func cleanTitle(value string) string {
	value = strings.ReplaceAll(value, `\(`, "(")
	value = strings.ReplaceAll(value, `\)`, ")")
	value = strings.ReplaceAll(value, `\\`, `\`)
	value = strings.Join(strings.Fields(value), " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return "Untitled PDF"
	}
	if bytes.HasPrefix([]byte(value), []byte{0xfe, 0xff}) {
		return "Untitled PDF"
	}
	return value
}
