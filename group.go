package spisql

type GroupOp string

const (
	GroupOr  GroupOp = "or"
	GroupAnd GroupOp = "and"
)

type Group struct {
	Op      GroupOp
	Filters Filters
}

type Groups []*Group
