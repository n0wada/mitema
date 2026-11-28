# Go parser

This repository contains a Go parser for the Mitema syntax described in
[SPEC.md](../SPEC.md). It uses only the Go standard library.

```go
file, err := mitema.Parse("[[航海]]\n船長 -> 船")
if err != nil { /* handle syntax error */ }
_ = file
```

`Parse` returns a syntax tree in source order, with positions on entries and
annotations. Relations retain their terms and operators, including lists,
chains, labels, negation and uncertainty. Comments are available both on their
associated nodes and in `File.Annotations` (including unattached comments).
Definitions and repeated top-level and section markers remain in source order;
consumers can build merged or last-definition-wins views from the tree.

From the repository root, run `go -C parser test ./...`.
