package spisql

type Column string

func NewColumn(s string) Column { return Column(s) }

func (c Column) String() string { return string(c) }
