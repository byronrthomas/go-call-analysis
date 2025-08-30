package graphcommon

type Mappable interface {
	ToMap() map[string]any
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
