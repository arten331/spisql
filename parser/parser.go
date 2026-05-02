package parser

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/arten331/spisql"
)

const (
	KeySort   = "sort"
	KeyLimit  = "limit"
	KeyOffset = "offset"
)

const (
	defaultLimit uint64 = 10
	defaultMax   uint64 = 100
)

type Options struct {
	Filter       FilterOptions
	Sort         SortOptions
	DefaultLimit uint64
	MaxLimit     uint64
}

type FilterOptions map[string]FilterOption
type SortOptions map[string]SortOption

type FilterOption struct{}
type SortOption struct{}

var bracketKey = regexp.MustCompile(`^([^\[\]]+)\[([^\[\]]+)\]$`)

func Parse(values url.Values, opts Options) *spisql.Query {
	return &spisql.Query{
		Filters: parseFilters(values, opts.Filter),
		Sorts:   parseSorts(values, opts.Sort),
		Pagination: spisql.Pagination{
			Limit:  parseLimit(values.Get(KeyLimit), opts),
			Offset: parseOffset(values.Get(KeyOffset)),
		},
	}
}

type filterEntry struct {
	raw, col string
	op       spisql.Operator
}

func parseFilters(values url.Values, allowed FilterOptions) spisql.Filters {
	entries := make([]filterEntry, 0, len(values))
	for k := range values {
		if k == KeySort || k == KeyLimit || k == KeyOffset || k == "" {
			continue
		}
		if m := bracketKey.FindStringSubmatch(k); m != nil {
			entries = append(entries, filterEntry{raw: k, col: m[1], op: spisql.Operator(m[2])})
			continue
		}
		if !strings.ContainsAny(k, "[]") {
			entries = append(entries, filterEntry{raw: k, col: k, op: spisql.OpEqual})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].raw < entries[j].raw })

	out := make(spisql.Filters, 0, len(entries))
	for _, e := range entries {
		if allowed != nil {
			if _, ok := allowed[e.col]; !ok {
				continue
			}
		}
		f := &spisql.Filter{
			Key:      spisql.NewColumn(e.col),
			Operator: e.op,
		}
		if !e.op.IsNullary() {
			f.Values = values[e.raw]
		}
		out = append(out, f)
	}
	return out
}

func parseSorts(values url.Values, allowed SortOptions) spisql.Sorts {
	out := make(spisql.Sorts, 0)
	for _, raw := range values[KeySort] {
		for tok := range strings.SplitSeq(raw, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			dir := spisql.DirectionAsc
			switch tok[0] {
			case '-':
				dir = spisql.DirectionDesc
				tok = tok[1:]
			case '+':
				tok = tok[1:]
			}
			if tok == "" {
				continue
			}
			if allowed != nil {
				if _, ok := allowed[tok]; !ok {
					continue
				}
			}
			out = append(out, &spisql.Sort{Key: spisql.NewColumn(tok), Direction: dir})
		}
	}
	return out
}

func parseLimit(s string, opts Options) uint64 {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 {
		if opts.DefaultLimit == 0 {
			return defaultLimit
		}
		return opts.DefaultLimit
	}
	max := opts.MaxLimit
	if max == 0 {
		max = defaultMax
	}
	if n > max {
		return max
	}
	return n
}

func parseOffset(s string) uint64 {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
