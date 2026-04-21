package agent

// FileChange is a provider-neutral description of one file's modifications
// inside a patch-approval request. Providers translate their wire-level
// patch representation into a slice of these so the UI can render a diff
// without knowing the protocol.
type FileChange struct {
	Path     string
	Op       FileChangeOp
	MovePath string
	Hunks    []DiffHunk
	Added    int
	Removed  int
}

// FileChangeOp enumerates the kinds of per-file operations a patch can
// contain. The UI uses this to shape headers (e.g. "added", "renamed").
type FileChangeOp string

const (
	FileChangeAdd    FileChangeOp = "add"
	FileChangeUpdate FileChangeOp = "update"
	FileChangeDelete FileChangeOp = "delete"
)

// DiffHunk is a single @@ -a,b +c,d @@ section of a unified diff. Header
// is the raw "@@ ..." line (empty for synthesized add/delete hunks).
type DiffHunk struct {
	Header string
	Lines  []DiffLine
}

// DiffLineKind classifies a single diff line. Context is an unchanged line,
// Add is a "+" line, Remove is a "-" line.
type DiffLineKind byte

const (
	DiffLineContext DiffLineKind = ' '
	DiffLineAdd     DiffLineKind = '+'
	DiffLineRemove  DiffLineKind = '-'
)

// DiffLine is one line inside a DiffHunk. Text never includes the leading
// sign character — Kind carries that information.
type DiffLine struct {
	Kind DiffLineKind
	Text string
}
