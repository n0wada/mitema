// Package mitema parses Mitema definition files into a syntax tree.
package mitema

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Position identifies a location in the input (lines and columns start at one).
type Position struct{ Line, Column int }
type Annotation struct {
	Text     string
	Position Position
}
type Metadata struct {
	Name, Value string
	Annotations []Annotation
}
type Name struct{ Parts []string }
type Argument struct {
	Name      string
	Uncertain bool
}
type Atom struct {
	Name      Name
	Arguments []Argument
}
type Item struct {
	Atoms       []Atom
	Uncertain   bool
	Annotations []Annotation
}
type Term struct {
	Items []Item
	List  bool
}
type Operator struct {
	Kind, Label string
	Negated     bool
}
type Relation struct {
	Terms     []Term
	Operators []Operator
}

// Entry represents a definition, relation, application, or mapping.
type Entry struct {
	Kind          string
	Metadata      []Metadata
	Annotations   []Annotation
	Left          Term
	Description   string
	Relation      *Relation
	Right         Term
	RightRelation *Relation
	Children      []Entry
	Negated       bool
	Position      Position
}
type Section struct {
	Path        Name
	Metadata    []Metadata
	Annotations []Annotation
	Entries     []Entry
}
type TopLevel struct {
	Name        string
	Metadata    []Metadata
	Annotations []Annotation
	Entries     []Entry
	Sections    []Section
}
type Document struct {
	Header *string
	Tops   []TopLevel
}

// File retains unattached comments as well as comments attached to syntax nodes.
type File struct {
	Documents   []Document
	Annotations []Annotation
}

type token struct {
	text, kind string
	pos        Position
	end        int
}
type parser struct {
	src            string
	off, line, col int
	tok            token
	comments       []Annotation
}

func (p *parser) errorf(format string, a ...any) error {
	return fmt.Errorf("%d:%d: %s", p.tok.pos.Line, p.tok.pos.Column, fmt.Sprintf(format, a...))
}
func (p *parser) advance(n int) {
	for _, r := range p.src[p.off : p.off+n] {
		if r == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
	}
	p.off += n
}
func (p *parser) next() error {
	for p.off < len(p.src) && (p.src[p.off] == ' ' || p.src[p.off] == '\t') {
		p.advance(1)
	}
	pos := Position{p.line, p.col}
	start := p.off
	if start == len(p.src) {
		p.tok = token{kind: "eof", pos: pos, end: start}
		return nil
	}
	c := p.src[start]
	if c == '\r' { // Parse validates CRLF even in raw header and definition text.
		p.advance(2)
		p.tok = token{text: "\n", kind: "nl", pos: pos, end: p.off}
		return nil
	}
	if c == '\n' {
		p.advance(1)
		p.tok = token{text: "\n", kind: "nl", pos: pos, end: p.off}
		return nil
	}
	if c == '#' {
		for p.off < len(p.src) && p.src[p.off] != '\n' && p.src[p.off] != '\r' {
			_, n := utf8.DecodeRuneInString(p.src[p.off:])
			p.advance(n)
		}
		v := p.src[start+1 : p.off]
		p.comments = append(p.comments, Annotation{v, pos})
		p.tok = token{text: v, kind: "comment", pos: pos, end: p.off}
		return nil
	}
	if c == '\'' {
		p.advance(1)
		var b strings.Builder
		for p.off < len(p.src) {
			if p.src[p.off] == '\n' || p.src[p.off] == '\r' {
				return fmt.Errorf("%d:%d: 改行を含む引用識別子", pos.Line, pos.Column)
			}
			if p.src[p.off] == '\'' {
				p.advance(1)
				if p.off < len(p.src) && p.src[p.off] == '\'' {
					b.WriteByte('\'')
					p.advance(1)
					continue
				}
				if b.Len() == 0 {
					return fmt.Errorf("%d:%d: 空の識別子", pos.Line, pos.Column)
				}
				p.tok = token{text: b.String(), kind: "quoted", pos: pos, end: p.off}
				return nil
			}
			_, n := utf8.DecodeRuneInString(p.src[p.off:])
			b.WriteString(p.src[p.off : p.off+n])
			p.advance(n)
		}
		return fmt.Errorf("%d:%d: 引用識別子が閉じられていません", pos.Line, pos.Column)
	}
	for _, op := range []string{"[[", "]]", "---", "-->", "==>", "-[", "=[", "<>", "->", "=>", ":>"} {
		if strings.HasPrefix(p.src[start:], op) {
			p.advance(len(op))
			p.tok = token{text: op, kind: "symbol", pos: pos, end: p.off}
			return nil
		}
	}
	if strings.ContainsRune("[](){}.,+?:=@\">", rune(c)) {
		p.advance(1)
		p.tok = token{text: string(c), kind: "symbol", pos: pos, end: p.off}
		return nil
	}
	for p.off < len(p.src) {
		r, n := utf8.DecodeRuneInString(p.src[p.off:])
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' || strings.ContainsRune("[](){}.,+?:=@\"#<>-", r) {
			break
		}
		p.advance(n)
	}
	if p.off == start {
		return fmt.Errorf("%d:%d: 不正な文字 %q", pos.Line, pos.Column, c)
	}
	p.tok = token{text: p.src[start:p.off], kind: "name", pos: pos, end: p.off}
	return nil
}

// syntax returns the spelling only for tokens that can act as grammar markers.
func (t token) syntax() string {
	if t.kind == "symbol" || t.kind == "name" {
		return t.text
	}
	return ""
}
func (p *parser) adjacent() bool {
	i := p.tok.end - len(p.tok.text)
	return i > 0 && p.src[i-1] != ' ' && p.src[i-1] != '\t' && p.src[i-1] != '\n' && p.src[i-1] != '\r'
}
func (p *parser) expect(s string) error {
	if p.tok.syntax() != s {
		return p.errorf("%q が必要です", s)
	}
	return p.next()
}
func (p *parser) name() (string, error) {
	if p.tok.kind != "name" && p.tok.kind != "quoted" {
		return "", p.errorf("名前が必要です")
	}
	s := p.tok.text
	return s, p.next()
}
func (p *parser) path() (Name, error) {
	var n Name
	s, e := p.name()
	if e != nil {
		return n, e
	}
	n.Parts = append(n.Parts, s)
	for p.tok.syntax() == "." {
		if e = p.next(); e != nil {
			return n, e
		}
		s, e = p.name()
		if e != nil {
			return n, e
		}
		n.Parts = append(n.Parts, s)
	}
	return n, nil
}
func (p *parser) atom() (Atom, error) {
	var a Atom
	var e error
	a.Name, e = p.path()
	if e != nil {
		return a, e
	}
	if p.tok.syntax() == "(" {
		if len(a.Name.Parts) != 1 {
			return a, p.errorf("適用名に階層は使えません")
		}
		if e = p.next(); e != nil {
			return a, e
		}
		for {
			s, err := p.name()
			if err != nil {
				return a, err
			}
			q := false
			if p.tok.syntax() == "?" {
				if !p.adjacent() {
					return a, p.errorf("? は名前に密着させてください")
				}
				q = true
				if err = p.next(); err != nil {
					return a, err
				}
			}
			a.Arguments = append(a.Arguments, Argument{s, q})
			if p.tok.syntax() != "," {
				break
			}
			if err = p.next(); err != nil {
				return a, err
			}
		}
		e = p.expect(")")
	}
	return a, e
}
func (p *parser) item(right bool) (Item, error) {
	var i Item
	a, e := p.atom()
	if e != nil {
		return i, e
	}
	i.Atoms = append(i.Atoms, a)
	for p.tok.syntax() == "+" {
		if e = p.next(); e != nil {
			return i, e
		}
		a, e = p.atom()
		if e != nil {
			return i, e
		}
		i.Atoms = append(i.Atoms, a)
	}
	if p.tok.syntax() == "?" {
		if !p.adjacent() {
			return i, p.errorf("? は名前に密着させてください")
		}
		if !right || len(i.Atoms) != 1 || len(i.Atoms[0].Arguments) > 0 {
			return i, p.errorf("この項に ? は付けられません")
		}
		i.Uncertain = true
		e = p.next()
	}
	return i, e
}
func (p *parser) term(right bool) (Term, error) {
	var t Term
	var e error
	if p.tok.syntax() != "[" {
		i, err := p.item(right)
		t.Items = append(t.Items, i)
		return t, err
	}
	t.List = true
	if e = p.next(); e != nil {
		return t, e
	}
	if right && p.tok.kind == "comment" {
		return t, p.errorf("リストの開始行に注釈は書けません")
	}
	for {
		var pending []Annotation
		if right {
			pending, e = p.listGap()
			if e != nil {
				return t, e
			}
		}
		if p.tok.syntax() == "]" {
			if len(pending) > 0 {
				return t, p.errorf("閉じ括弧の直前に独立した注釈は書けません")
			}
			if len(t.Items) > 0 {
				return t, p.errorf("コンマの後に項が必要です")
			}
			e = p.next()
			return t, e
		}
		i, err := p.item(right)
		if err != nil {
			return t, err
		}
		i.Annotations = pending
		t.Items = append(t.Items, i)
		if right && p.tok.kind == "comment" {
			t.Items[len(t.Items)-1].Annotations = append(t.Items[len(t.Items)-1].Annotations, Annotation{p.tok.text, p.tok.pos})
			if e = p.next(); e != nil {
				return t, e
			}
		}
		if right {
			notes, err := p.listGap()
			if err != nil {
				return t, err
			}
			if len(notes) > 0 {
				return t, p.errorf("独立した注釈の直下には項が必要です")
			}
		}
		if p.tok.syntax() == "," {
			if e = p.next(); e != nil {
				return t, e
			}
			if right && p.tok.kind == "comment" {
				t.Items[len(t.Items)-1].Annotations = append(t.Items[len(t.Items)-1].Annotations, Annotation{p.tok.text, p.tok.pos})
				if e = p.next(); e != nil {
					return t, e
				}
			}
			if p.tok.syntax() == "]" {
				return t, p.errorf("コンマの後に項が必要です")
			}
			continue
		}
		if p.tok.syntax() != "]" {
			return t, p.errorf("リストの区切りが必要です")
		}
		e = p.next()
		return t, e
	}
}

// listGap consumes line breaks and returns comments directly above the next token.
// Comments separated by a blank line remain available in File.Annotations.
func (p *parser) listGap() ([]Annotation, error) {
	var notes []Annotation
	for p.tok.kind == "nl" {
		if err := p.next(); err != nil {
			return nil, err
		}
		switch p.tok.kind {
		case "nl":
			notes = nil
		case "comment":
			notes = append(notes, Annotation{p.tok.text, p.tok.pos})
			if err := p.next(); err != nil {
				return nil, err
			}
		}
	}
	return notes, nil
}
func (p *parser) op() (Operator, bool, error) {
	var o Operator
	if p.tok.syntax() == "not" {
		o.Negated = true
		if e := p.next(); e != nil {
			return o, false, e
		}
	}
	switch p.tok.syntax() {
	case ">", "<>", "->", "=>", ":>", ":", "-->", "==>":
		o.Kind = p.tok.text
		return o, true, p.next()
	case "-[", "=[":
		start := p.tok.text
		o.Kind = "-->"
		end := "->"
		if start == "=[" {
			o.Kind = "==>"
			end = "=>"
		}
		if e := p.next(); e != nil {
			return o, false, e
		}
		if p.tok.syntax() != "]" {
			s, e := p.name()
			if e != nil {
				return o, false, e
			}
			o.Label = s
		}
		if e := p.expect("]"); e != nil {
			return o, false, e
		}
		if !p.adjacent() {
			return o, false, p.errorf("矢印の終端に空白は使えません")
		}
		if e := p.expect(end); e != nil {
			return o, false, e
		}
		return o, true, nil
	}
	if o.Negated {
		return o, false, p.errorf("not の後に演算子が必要です")
	}
	return o, false, nil
}
func (p *parser) relation(left Term) (Relation, error) {
	r := Relation{Terms: []Term{left}}
	for {
		o, ok, e := p.op()
		if e != nil {
			return r, e
		}
		if !ok {
			break
		}
		if len(r.Operators) > 0 && (o.Kind != r.Operators[0].Kind || r.Terms[len(r.Terms)-1].List || len(r.Terms[len(r.Terms)-1].Items) != 1) {
			return r, p.errorf("異なる演算子や列挙を連鎖できません")
		}
		t, e := p.term(true)
		if e != nil {
			return r, e
		}
		r.Operators = append(r.Operators, o)
		r.Terms = append(r.Terms, t)
	}
	if len(r.Operators) == 0 {
		return r, p.errorf("関係演算子が必要です")
	}
	if len(r.Operators) > 1 && r.Terms[len(r.Terms)-1].List {
		return r, p.errorf("連鎖にリストは使えません")
	}
	if len(r.Operators) == 1 {
		for p.tok.syntax() == "," {
			if e := p.next(); e != nil {
				return r, e
			}
			i, e := p.item(true)
			if e != nil {
				return r, e
			}
			r.Terms[1].Items = append(r.Terms[1].Items, i)
			r.Terms[1].List = true
		}
	}
	return r, nil
}
func (p *parser) description() (string, error) { // Current token is '='; read raw text before lexing it.
	p.off = p.tok.end
	for p.off < len(p.src) && (p.src[p.off] == ' ' || p.src[p.off] == '\t') {
		p.advance(1)
	}
	if p.off == len(p.src) {
		return "", p.errorf("説明が必要です")
	}
	var s string
	if p.src[p.off] == '\'' || p.src[p.off] == '"' {
		q := p.src[p.off]
		p.advance(1)
		var b strings.Builder
		closed := false
		for p.off < len(p.src) {
			if p.src[p.off] == q {
				p.advance(1)
				if p.off < len(p.src) && p.src[p.off] == q {
					b.WriteByte(q)
					p.advance(1)
					continue
				}
				closed = true
				break
			}
			_, n := utf8.DecodeRuneInString(p.src[p.off:])
			b.WriteString(p.src[p.off : p.off+n])
			p.advance(n)
		}
		if !closed {
			return "", p.errorf("説明の引用符が閉じられていません")
		}
		s = b.String()
	} else {
		start := p.off
		for p.off < len(p.src) && p.src[p.off] != '\n' && p.src[p.off] != '\r' && p.src[p.off] != '#' {
			_, n := utf8.DecodeRuneInString(p.src[p.off:])
			p.advance(n)
		}
		s = strings.Trim(p.src[start:p.off], " \t")
	}
	if s == "" {
		return "", p.errorf("空の説明は使えません")
	}
	return s, p.next()
}
func (p *parser) entry() (Entry, error) {
	e := Entry{Position: p.tok.pos}
	var err error
	if p.tok.syntax() == "not" {
		e.Negated = true
		if err = p.next(); err != nil {
			return e, err
		}
	}
	e.Left, err = p.term(false)
	if err != nil {
		return e, err
	}
	if p.tok.syntax() == "=" {
		if e.Negated {
			return e, p.errorf("定義を否定できません")
		}
		e.Kind = "definition"
		e.Description, err = p.description()
		return e, err
	}
	if p.tok.syntax() == "map" {
		if e.Negated {
			return e, p.errorf("対応文を否定できません")
		}
		e.Kind = "mapping"
		if err = p.next(); err != nil {
			return e, err
		}
		e.Right, err = p.term(false)
		return e, err
	}
	if p.tok.syntax() == "not" || isOp(p.tok.syntax()) {
		if e.Negated {
			return e, p.errorf("関係の否定は演算子の直前に置いてください")
		}
		var r Relation
		r, err = p.relation(e.Left)
		if err != nil {
			return e, err
		}
		e.Kind = "relation"
		e.Relation = &r
		if p.tok.syntax() == "map" {
			e.Kind = "mapping"
			if err = p.next(); err != nil {
				return e, err
			}
			var t Term
			t, err = p.term(false)
			if err != nil {
				return e, err
			}
			var rr Relation
			rr, err = p.relation(t)
			e.RightRelation = &rr
		}
		return e, err
	}
	if e.Left.List || len(e.Left.Items) != 1 || len(e.Left.Items[0].Atoms) != 1 || len(e.Left.Items[0].Atoms[0].Arguments) == 0 {
		return e, p.errorf("単独の適用文が必要です")
	}
	e.Kind = "application"
	return e, nil
}

// mapChild follows the narrower term -> term grammar inside a mapping.
func (p *parser) mapChild() (Entry, error) {
	entry := Entry{Kind: "relation", Position: p.tok.pos}
	left, err := p.term(false)
	if err != nil {
		return entry, err
	}
	if err = p.expect("->"); err != nil {
		return entry, err
	}
	right, err := p.term(false)
	if err != nil {
		return entry, err
	}
	entry.Left = left
	entry.Relation = &Relation{Terms: []Term{left, right}, Operators: []Operator{{Kind: "->"}}}
	return entry, nil
}
func isOp(s string) bool {
	switch s {
	case ">", "<>", "->", "=>", ":>", ":", "-->", "==>", "-[", "=[":
		return true
	}
	return false
}
func (p *parser) metadata() (Metadata, error) {
	var m Metadata
	var e error
	nameColumn := p.tok.pos.Column + 1
	if e = p.expect("@"); e != nil {
		return m, e
	}
	if p.tok.pos.Column != nameColumn {
		return m, p.errorf("@ の直後に名前が必要です")
	}
	m.Name, e = p.name()
	if e != nil {
		return m, e
	}
	if !p.adjacent() {
		return m, p.errorf("名前の直後に ( が必要です")
	}
	if e = p.expect("("); e != nil {
		return m, e
	}
	m.Value, e = p.name()
	if e != nil {
		return m, e
	}
	e = p.expect(")")
	return m, e
}
func (p *parser) lineEnd() error {
	if p.tok.kind == "comment" {
		if e := p.next(); e != nil {
			return e
		}
	}
	if p.tok.kind == "nl" {
		return p.next()
	}
	if p.tok.kind != "eof" {
		return p.errorf("行末が必要です")
	}
	return nil
}

// Parse parses UTF-8 Mitema source. Errors include a one-based source position.
func Parse(src string) (*File, error) {
	if !utf8.ValidString(src) {
		return nil, fmt.Errorf("入力が UTF-8 ではありません")
	}
	for offset := 0; offset < len(src); {
		i := strings.IndexByte(src[offset:], '\r')
		if i < 0 {
			break
		}
		i += offset
		if i+1 == len(src) || src[i+1] != '\n' {
			line := strings.Count(src[:i], "\n") + 1
			start := strings.LastIndexByte(src[:i], '\n') + 1
			return nil, fmt.Errorf("%d:%d: 単独の CR は使えません", line, utf8.RuneCountInString(src[start:i])+1)
		}
		offset = i + 2
	}
	p := &parser{src: src, line: 1, col: 1}
	f := &File{Documents: []Document{{}}}
	if e := p.next(); e != nil {
		return nil, e
	}
	var top *TopLevel
	var sec *Section
	var pending []Annotation
	var meta []Metadata
	var block, parentView *Entry
	for p.tok.kind != "eof" {
		if len(meta) > 0 && (p.tok.kind == "nl" || p.tok.kind == "comment") {
			return nil, p.errorf("メタデータの直後に対象が必要です")
		}
		if p.tok.kind == "nl" {
			pending = nil
			if e := p.next(); e != nil {
				return nil, e
			}
			continue
		}
		if p.tok.kind == "comment" {
			pending = append(pending, Annotation{p.tok.text, p.tok.pos})
			if e := p.lineEnd(); e != nil {
				return nil, e
			}
			continue
		}
		// A delimiter is recognized only as a complete line, and header text is raw.
		lineStart := strings.LastIndex(src[:p.off-len(p.tok.text)], "\n") + 1
		lineEnd := strings.IndexByte(src[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(src)
		} else {
			lineEnd += lineStart
		}
		if strings.Trim(src[lineStart:lineEnd], " \t\r") == "---" {
			if block != nil || len(meta) > 0 {
				return nil, p.errorf("この位置にヘッダは置けません")
			}
			start := lineEnd
			if start < len(src) {
				start++
			}
			end := start
			found := false
			for end <= len(src) {
				j := strings.IndexByte(src[end:], '\n')
				stop := len(src)
				if j >= 0 {
					stop = end + j
				}
				if strings.Trim(src[end:stop], " \t\r") == "---" {
					found = true
					break
				}
				if j < 0 {
					break
				}
				end = stop + 1
			}
			if !found {
				return nil, p.errorf("ヘッダが閉じられていません")
			}
			h := src[start:end]
			if len(f.Documents) == 1 && len(f.Documents[0].Tops) == 0 && f.Documents[0].Header == nil {
				f.Documents[0].Header = &h
			} else {
				f.Documents = append(f.Documents, Document{Header: &h})
			}
			p.advance(end - p.off)
			for p.off < len(src) && src[p.off] != '\n' {
				p.advance(1)
			}
			if p.off < len(src) {
				p.advance(1)
			}
			if e := p.next(); e != nil {
				return nil, e
			}
			top = nil
			sec = nil
			pending = nil
			continue
		}
		if p.tok.syntax() == "@" {
			m, e := p.metadata()
			if e != nil {
				return nil, e
			}
			m.Annotations = pending
			pending = nil
			if p.tok.kind == "comment" {
				m.Annotations = append(m.Annotations, Annotation{p.tok.text, p.tok.pos})
			}
			meta = append(meta, m)
			if e = p.lineEnd(); e != nil {
				return nil, e
			}
			continue
		}
		if p.tok.syntax() == "[[" {
			if block != nil {
				return nil, p.errorf("ブロック内でトップレベルを変更できません")
			}
			if e := p.next(); e != nil {
				return nil, e
			}
			n, e := p.name()
			if e != nil {
				return nil, e
			}
			if e = p.expect("]]"); e != nil {
				return nil, e
			}
			d := &f.Documents[len(f.Documents)-1]
			d.Tops = append(d.Tops, TopLevel{Name: n, Metadata: meta, Annotations: pending})
			top = &d.Tops[len(d.Tops)-1]
			sec = nil
			meta = nil
			pending = nil
			if p.tok.kind == "comment" {
				top.Annotations = append(top.Annotations, Annotation{p.tok.text, p.tok.pos})
			}
			if e = p.lineEnd(); e != nil {
				return nil, e
			}
			continue
		}
		if top == nil {
			return nil, p.errorf("本文には [[名前]] が必要です")
		}
		if p.tok.syntax() == "}" {
			if block == nil || len(meta) > 0 {
				return nil, p.errorf("対応するブロックがありません")
			}
			if len(pending) > 0 {
				return nil, p.errorf("閉じ括弧の直前に独立した注釈は書けません")
			}
			if e := p.next(); e != nil {
				return nil, e
			}
			if p.tok.kind == "comment" {
				block.Annotations = append(block.Annotations, Annotation{p.tok.text, p.tok.pos})
			}
			block, parentView = parentView, nil
			if e := p.lineEnd(); e != nil {
				return nil, e
			}
			continue
		}
		// A section is a bracketed path on a line by itself; otherwise it is a list term.
		if p.tok.syntax() == "[" && block == nil {
			save := *p
			if e := p.next(); e != nil {
				return nil, e
			}
			path, e := p.path()
			if e == nil && p.tok.syntax() == "]" {
				e = p.next()
				if e == nil && (p.tok.kind == "nl" || p.tok.kind == "comment" || p.tok.kind == "eof") {
					top.Sections = append(top.Sections, Section{Path: path, Metadata: meta, Annotations: pending})
					sec = &top.Sections[len(top.Sections)-1]
					if p.tok.kind == "comment" {
						sec.Annotations = append(sec.Annotations, Annotation{p.tok.text, p.tok.pos})
					}
					meta = nil
					pending = nil
					if e = p.lineEnd(); e != nil {
						return nil, e
					}
					continue
				}
			}
			*p = save
		}
		var entry Entry
		var e error
		if block != nil && block.Kind == "mapping" {
			entry, e = p.mapChild()
			if e != nil {
				return nil, e
			}
		} else if p.tok.syntax() == "view" {
			if block != nil {
				return nil, p.errorf("ブロックの入れ子は使えません")
			}
			entry = Entry{Kind: "view", Position: p.tok.pos}
			if e = p.next(); e != nil {
				return nil, e
			}
			n, err := p.name()
			if err != nil {
				return nil, err
			}
			entry.Left = Term{Items: []Item{{Atoms: []Atom{{Name: Name{Parts: []string{n}}}}}}}
			if e = p.expect("{"); e != nil {
				return nil, e
			}
		} else {
			entry, e = p.entry()
			if e != nil {
				return nil, e
			}
		}
		entry.Metadata = meta
		entry.Annotations = pending
		meta = nil
		pending = nil
		opens := entry.Kind == "view"
		if entry.Kind == "mapping" && p.tok.syntax() == "{" {
			if entry.RightRelation != nil {
				return nil, p.errorf("この対応にはブロックを置けません")
			}
			opens = true
			if e = p.next(); e != nil {
				return nil, e
			}
		}
		if p.tok.kind == "comment" {
			if opens {
				return nil, p.errorf("ブロックの開始行に注釈は書けません")
			}
			entry.Annotations = append(entry.Annotations, Annotation{p.tok.text, p.tok.pos})
		}
		if block != nil {
			block.Children = append(block.Children, entry)
			if opens {
				parentView = block
				block = &block.Children[len(block.Children)-1]
			}
		} else if sec != nil {
			sec.Entries = append(sec.Entries, entry)
			if opens {
				block = &sec.Entries[len(sec.Entries)-1]
			}
		} else {
			top.Entries = append(top.Entries, entry)
			if opens {
				block = &top.Entries[len(top.Entries)-1]
			}
		}
		if e = p.lineEnd(); e != nil {
			return nil, e
		}
	}
	if block != nil {
		return nil, p.errorf("ブロックが閉じられていません")
	}
	if len(meta) > 0 {
		return nil, p.errorf("メタデータの対象がありません")
	}
	f.Annotations = p.comments
	return f, nil
}
