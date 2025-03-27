package selector

import "errors"

var (
	// ErrNoAvailable 是没有可用节点的错误
	ErrNoAvailable = errors.New("no available node")
)
