package doc

import (
	//	"fmt"
	"strings"
	
	"github.com/boynton/api/model"
	//	"github.com/boynton/api/smithy"
	//	"github.com/boynton/data"
)

type MarkdownFormat struct {
	gen *Generator
}
func (m *MarkdownFormat) FileExtension() string {
	return ".md"
}

func (m *MarkdownFormat) RenderHeader() {
}
func (m *MarkdownFormat) RenderSummary() {
	m.gen.Emitf("\n# %s\n\n", model.Capitalize(m.gen.name))
	m.gen.Emit(model.FormatComment("", "", m.gen.Schema.Comment, 80, true))
	m.gen.Emit("\n")
	if m.gen.name != "" {
		m.gen.Emitf("- **Service**: %q\n", m.gen.name)
	}
	if m.gen.Schema.Version != "" {
		m.gen.Emitf("- **Version**: %q\n", m.gen.Schema.Version)
	}
	if m.gen.ns != "" {
		m.gen.Emitf("- **Namespace**: %q\n", m.gen.ns)
	}
	if m.gen.Schema.Base != "" {
		m.gen.Emitf("- **Base: %s\n", m.gen.Schema.Base)
	}
	m.gen.Emitf("\n")
	rezIds := m.gen.ResourceIds()
	if len(rezIds) > 0 {
		m.gen.Emitf("### Resource Index\n")
		for _, id := range rezIds {
			s := StripNamespace(id)
			m.gen.Emitf("- [%s](%s)\n", s, strings.ToLower(s))
		}
		m.gen.Emitf("\n")
	}
	opIds := m.gen.Operations()
	if len(opIds) > 0 {
		m.gen.Emitf("### Operation Index\n")
		for _, op := range opIds {
			sum := summarySignature(op)
			s := StripNamespace(op.Id)
			m.gen.Emitf("- [%s](#%s)\n", sum, strings.ToLower(s))
		}
		m.gen.Emitf("\n")
	}
	m.gen.Emit("\n### Type Index\n\n")
	for _, td := range m.gen.Types() {
		if strings.HasPrefix(string(td.Id), "aws.protocols#") || strings.HasPrefix(string(td.Id), "smithy.api#"){
			continue
		}
		s := StripNamespace(td.Id)
		m.gen.Emitf("- [%s](#%s) → _%s_\n", s, strings.ToLower(s), td.Base)
	}
	m.gen.Emitf("\n\n")
}

func (m *MarkdownFormat) RenderFooter() {
}
