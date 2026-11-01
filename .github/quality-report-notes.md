# Interpreting Go Report Card

The pinned report tool runs an English spelling checker recursively over the
repository. This includes translated Markdown documents and locale catalogs.
Its English replacements are not corrections for French, German or Italian
prose. Some accented words are split into English-looking fragments by the
checker. Review those suggestions in the document's language before editing.

The report retains every raw finding and its score. The annotation script adds
this language-scope explanation without changing scores, excluding files or
disabling checks. English source and documentation remain subject to the same
spelling checks.

In this tool, the spelling score has zero weight in the overall grade. An A+
grade of 100% therefore does not mean that the raw findings list is empty.

Remove the previous generated report before running the tool, as CI does.
Otherwise the checker also scans words quoted in that report and reports the
report itself. The committed report is regenerated with the pinned v1.1.4 Linux
tools after the complexity refactor; it is not a live view of every later commit.
CI artifacts contain the report generated for each checked commit.

The v1.1.4 runner also skips its standalone ineffassign command on large
repositories. The separate golangci-lint job still enables ineffassign for the
whole Go module. A Go Report Card grade alone is not evidence that every tool
ran or that translated text has been checked by a native-language dictionary.
