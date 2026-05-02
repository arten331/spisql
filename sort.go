package spisql

type Direction string

const (
	DirectionAsc  Direction = "ASC"
	DirectionDesc Direction = "DESC"
)

func (d Direction) String() string { return string(d) }

type Sort struct {
	Key       Column
	Direction Direction
}

type Sorts []*Sort
