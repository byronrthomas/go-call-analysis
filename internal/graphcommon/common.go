package graphcommon

import "path/filepath"

type Mappable interface {
	ToMap() map[string]any
}

type NodeTypes struct {
	FromLabel string
	ToLabel   string
}

type EdgeMappable interface {
	ToMap() map[string]any
	NodeTypes() NodeTypes
}

type FileVersionNode struct {
	Id              string
	LastGitRevision string
}

type PositionInfo struct {
	Line   int
	Column int
}

type NodeCommon struct {
	ID           string
	Name         string
	Package      string
	PositionInfo PositionInfo
}

func (node FileVersionNode) ToMap() map[string]any {
	return map[string]any{
		"id":                node.Id,
		"name":              filepath.Base(node.Id),
		"last_git_revision": node.LastGitRevision,
		"label":             "FileVersion",
	}
}

func NodeCommonAsMap(nodeCommon NodeCommon) map[string]any {
	return map[string]any{
		"id":   nodeCommon.ID,
		"name": nodeCommon.Name,
		// "package": nodeCommon.Package,
		"line":   nodeCommon.PositionInfo.Line,
		"column": nodeCommon.PositionInfo.Column,
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
