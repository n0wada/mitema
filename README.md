# Mitema

[日本語](README.ja.md)

<p align="center">
  <img src="mitema.png" alt="Mitema emblem with interwoven warp and weft threads" width="120">
</p>

Mitema is a plain-text language for describing relationships between concepts. You can use it to record who did what in a story, what a symbol brings to mind, or how a relationship in one setting corresponds to a relationship in another.

The name comes from the Spanish word *mitema*, meaning “mytheme,” a basic unit of a myth.

## Basic operators

| Operator | Meaning | Example |
|---|---|---|
| `>` | The term on the left is a broader category. | `vessel > sailboat` |
| `<>` | Contrast two concepts. | `calm <> storm` |
| `->` | Link a concept to an association or attribute. | `compass -> direction` |
| `=>` | Read the left-hand term through the right-hand term as a metaphor. | `state => ship` |
| `-->`, `-[label]->` | Connect two terms in a relationship diagram. The optional label names the relationship. | `captain + navigator -[choose]-> course` |
| `==>`, `=[label]=>` | The term on the left becomes the one on the right. The optional label names the event or process. | `apprentice =[appointment]=> navigator` |
| `:>` | Describe a trait or behavior associated with a role. | `navigator :> 'plots the course'` |
| `:` | List one or more parts of the whole on the left. | `ship : [hull, sail]` |
| `=` | Define a term with a text description. | `captain = the person in command of a ship` |

## Writing a metaphor

If a state is a ship, who is its captain? Start the body of a text file with a name in double brackets, then write down the metaphor and the correspondences you see.

```text
[['Ship of state']]
state => ship

navigation map governance {
  captain -> ruler
  ship    -> state
}

captain -[steers]-> ship map ruler -[governs]-> state
```

Read `state => ship` as “think of the state as a ship.” Inside the `map` block, each arrow pairs a term from navigation with its counterpart in governance. The final line maps a whole relationship. The captain steers the ship as the ruler governs the state.

## Recording a point of view

Two people on the same voyage may give different accounts of what happened.

```text
[['Voyage log']]
view surgeon {
  captain -[consults]-> surgeon
}
view apprentice {
  captain -[consults]-> surgeon?
}
```

In the surgeon’s account, the captain consulted the surgeon. In the apprentice’s account, the trailing `?` leaves open whether the consultation took place. Each `view` records that person’s perspective explicitly.

## Specification and license

The definition file format is under development. The first version focuses on recording relationships. Search and extraction are left to separate tools.

- [Definition file specification (Japanese)](SPEC.md) — operators, sections, annotations, and the EBNF grammar.
- [MIT License](LICENSE)
