package spisql

type Operator string

const (
	OpEqual                   Operator = "eq"
	OpNotEqual                Operator = "ne"
	OpIn                      Operator = "in"
	OpNotIn                   Operator = "nin"
	OpGreaterThan             Operator = "gt"
	OpGreaterThanEqual        Operator = "gte"
	OpGreaterThanEqualISODate Operator = "gtedate"
	OpLessThan                Operator = "lt"
	OpLessThanEqual           Operator = "lte"
	OpSubstringOf             Operator = "substringof"
	OpStartsWith              Operator = "startswith"
	OpEndsWith                Operator = "endswith"
	OpIsNull                  Operator = "isnull"
	OpIsNotNull               Operator = "isnotnull"
	OpBetween                 Operator = "between"
	OpNotBetween              Operator = "nbetween"
)

func (o Operator) IsNullary() bool {
	return o == OpIsNull || o == OpIsNotNull
}

func (o Operator) String() string { return string(o) }

type Filter struct {
	Key      Column
	Operator Operator
	Values   []string
}

type Filters []*Filter

func (fs Filters) Add(f Filter) Filters { return append(fs, &f) }
