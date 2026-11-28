package mitema

import (
	"strings"
	"testing"
)

func TestLineEndings(t *testing.T) {
	for _, s := range []string{"[[X]]\r\n天秤\r->B", "[[X]]\nA -> B\r", "---\n原文\r\n---\r", "[[X]]\nA = \"a\rb\"", "[[X]]\n# 注\r釈"} {
		_, err := Parse(s)
		if err == nil || !strings.Contains(err.Error(), "単独の CR") {
			t.Errorf("Parse(%q): %v", s, err)
		}
	}
	if _, err := Parse("[[X]]\r\n天秤\r->B"); err == nil || !strings.HasPrefix(err.Error(), "2:3:") {
		t.Errorf("unexpected CR position: %v", err)
	}
	for _, nl := range []string{"\n", "\r\n"} {
		src := strings.Join([]string{"---", "原文", "---", "[[X]]", "A = \"a", "b\"", "A -> B # 注"}, nl)
		f, err := Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		d := f.Documents[0]
		entries := d.Tops[0].Entries
		if *d.Header != "原文"+nl || entries[0].Description != "a"+nl+"b" || entries[1].Position.Line != 7 {
			t.Errorf("unexpected newline handling for %q", nl)
		}
	}
}

func TestAnnotationOwnership(t *testing.T) {
	body := "[[X]]\nview V {\nA map B {\n# source\n@src(x)\nC -> D\n} # map\n} # view\nE -> [\n# detached\n\n# first\nF, # comma\nG # last\n] # list\n# eof"
	f, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	entries := f.Documents[0].Tops[0].Entries
	view, list := entries[0], entries[1]
	mapping := view.Children[0]
	items := list.Relation.Terms[1].Items
	check := func(notes []Annotation, want ...string) {
		t.Helper()
		if len(notes) != len(want) {
			t.Fatalf("annotations %v, want %v", notes, want)
		}
		for i, note := range notes {
			if note.Text != want[i] {
				t.Errorf("annotation %q, want %q", note.Text, want[i])
			}
		}
	}
	check(view.Annotations, " view")
	check(mapping.Annotations, " map")
	check(mapping.Children[0].Metadata[0].Annotations, " source")
	check(list.Annotations, " list")
	check(items[0].Annotations, " first", " comma")
	check(items[1].Annotations, " last")
	check(f.Annotations, " source", " map", " view", " detached", " first", " comma", " last", " list", " eof")
}

func TestAmbiguousAnnotations(t *testing.T) {
	for _, body := range []string{
		"A -> [ # open\nB]", "view V { # open\n}", "A map B { # open\n}",
		"A -> [\n# before close\n]", "A -> [B\n# before close\n]",
		"A -> [B\n# before comma\n, C]", "view V {\n# before close\n}", "A map B {\n# before close\n}",
		"@src(x)\n\nA -> B", "@src(x)\n# gap\nA -> B", "@src(x)\n\nview V {\n}",
	} {
		if _, err := Parse("[[X]]\n" + body); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}

func TestViewMetadata(t *testing.T) {
	for _, body := range []string{"", "A -> B\n"} {
		f, err := Parse("[[X]]\n@id('view_id')\nview V {\n" + body + "}\nC -> D")
		if err != nil {
			t.Fatal(err)
		}
		entries := f.Documents[0].Tops[0].Entries
		view := entries[0]
		if view.Kind != "view" || len(view.Metadata) != 1 || view.Metadata[0].Name != "id" || view.Metadata[0].Value != "view_id" {
			t.Fatalf("unexpected view metadata: %+v", view.Metadata)
		}
		if len(entries[1].Metadata) != 0 || (body != "" && len(view.Children[0].Metadata) != 0) {
			t.Fatal("view metadata attached to another entry")
		}
	}
}

func TestExample(t *testing.T) {
	f, err := Parse("---\n航海日誌\n---\n[[船]]\n[乗組員]\n船長 -> 船\n[航路]\n港 -> 海")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Documents) != 1 || f.Documents[0].Header == nil || len(f.Documents[0].Tops) != 1 || len(f.Documents[0].Tops[0].Sections) != 2 {
		t.Fatalf("unexpected document: %+v", f.Documents)
	}
}

func TestSyntax(t *testing.T) {
	source := "---\nheader # text\n---\n[[航海]]\n@source(log)\n[乗組員]\nview 船医 {\n A -[相談]-> B?\n}\n航海 map 統治 {\n 船 -> 国\n}\nA ==> B not =[儀式]=> C\n説明 = \"一行目\nC# と \"\"引用\"\"\" # 注\n"
	f, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	s := f.Documents[0].Tops[0].Sections[0]
	if len(s.Entries) != 4 || len(s.Entries[0].Children) != 1 || len(s.Entries[1].Children) != 1 {
		t.Fatalf("entries: %+v", s.Entries)
	}
	r := s.Entries[2].Relation
	if len(r.Operators) != 2 || !r.Operators[1].Negated || r.Operators[1].Label != "儀式" {
		t.Fatalf("relation: %+v", r)
	}
	if s.Entries[3].Description != "一行目\nC# と \"引用\"" {
		t.Fatalf("description: %q", s.Entries[3].Description)
	}
}

func TestInvalid(t *testing.T) {
	for _, s := range []string{"A -> B", "[[X]]\nA()", "[[X]]\nA -> B > C", "[[X]]\nA = ''", "[[X]]\nview X {\nA -> B", "[[X]]\nA -> [B, [C]]", "[[X]]\n@x(y)"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}

func TestReviewedGrammar(t *testing.T) {
	source := "[[X]]\n'view' = text\n[A]->B #->\nview V {\nA map B {\nC -> D\n}\nE -> [F\n, G]\n}\nH -> I"
	file, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	entries := file.Documents[0].Tops[0].Entries
	if len(entries) != 4 || entries[0].Left.Items[0].Atoms[0].Name.Parts[0] != "view" ||
		!entries[1].Left.List || len(entries[1].Annotations) != 1 || entries[1].Annotations[0].Text != "->" {
		t.Fatalf("unexpected top-level entries: %+v", entries)
	}
	view := entries[2].Children
	if len(view) != 2 || view[0].Kind != "mapping" || len(view[0].Children) != 1 ||
		len(view[1].Relation.Terms[1].Items) != 2 {
		t.Fatalf("unexpected view entries: %+v", view)
	}
	for _, body := range []string{"A -> [B,\n]", "not A map B", "@ x(y)\nA -> B", "@x (y)\nA -> B",
		"A map B {\nC -> D?\n}", "A map B {\nC -> D, E\n}", "A-[x] ->B"} {
		if _, err := Parse("[[X]]\n" + body); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}
