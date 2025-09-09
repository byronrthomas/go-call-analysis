package graphcommon

type Mappable interface {
	ToMap() map[string]any
}

type PositionInfo struct {
	File            string
	LastGitRevision string
	Line            int
	Column          int
}

type NodeCommon struct {
	ID           string
	Name         string
	Package      string
	PositionInfo PositionInfo
}

func NodeCommonAsMap(nodeCommon NodeCommon) map[string]any {
	return map[string]any{
		"id":                nodeCommon.ID,
		"name":              nodeCommon.Name,
		"package":           nodeCommon.Package,
		"file":              nodeCommon.PositionInfo.File,
		"last_git_revision": nodeCommon.PositionInfo.LastGitRevision,
		"line":              nodeCommon.PositionInfo.Line,
		"column":            nodeCommon.PositionInfo.Column,
	}
}

type EdgeCommon struct {
	FromID string
	ToID   string
}

func EdgeCommonAsMap(edgeCommon EdgeCommon) map[string]any {
	return map[string]any{
		"from_id": edgeCommon.FromID,
		"to_id":   edgeCommon.ToID,
	}
}
