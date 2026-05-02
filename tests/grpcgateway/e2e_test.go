//go:build integration

package grpcgateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	usersv1 "github.com/arten331/spisql/tests/grpcgateway/proto/usersv1"
)

func TestE2E_GRPCGateway(t *testing.T) {
	db := pg(t)
	st := startStack(t, db)
	t.Cleanup(st.close)

	cases := []struct {
		name      string
		filter    string // raw spisql wire fragment; client URL-encodes it
		sort      string
		limit     uint64
		offset    uint64
		wantTotal uint64
		wantPage  []string // expected names in order, after limit/offset/sort
	}{
		{
			name:      "eq",
			filter:    "name=vasya",
			wantTotal: 1,
			wantPage:  []string{"vasya"},
		},
		{
			name:      "in",
			filter:    "name[in]=vasya&name[in]=petya",
			sort:      "name",
			wantTotal: 2,
			wantPage:  []string{"petya", "vasya"},
		},
		{
			name:      "isnull",
			filter:    "email[isnull]",
			wantTotal: 1,
			wantPage:  []string{"masha"},
		},
		{
			name:      "between",
			filter:    "id[between]=2&id[between]=3",
			sort:      "id",
			wantTotal: 2,
			wantPage:  []string{"petya", "masha"},
		},
		{
			name:      "sort_desc_limit",
			sort:      "-name",
			limit:     2,
			wantTotal: 4,
			wantPage:  []string{"vasya", "petya"},
		},
		{
			name:      "offset_paging",
			sort:      "name",
			limit:     2,
			offset:    2,
			wantTotal: 4,
			wantPage:  []string{"petya", "vasya"},
		},
		{
			name:      "whitelist_drops_unknown_field",
			filter:    "secret=anything",
			wantTotal: 4,
		},
		{
			name:      "typed_overrides_filter_embedded",
			filter:    "sort=name&limit=99",
			sort:      "-name",
			limit:     1,
			wantTotal: 4,
			wantPage:  []string{"vasya"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := callList(t, st.httpURL, tc.filter, tc.sort, tc.limit, tc.offset)
			if resp.Total != tc.wantTotal {
				t.Errorf("total: want %d, got %d", tc.wantTotal, resp.Total)
			}
			if tc.wantPage == nil {
				return
			}
			if len(resp.Items) != len(tc.wantPage) {
				t.Fatalf("page len: want %d, got %d (%+v)", len(tc.wantPage), len(resp.Items), resp.Items)
			}
			for i, want := range tc.wantPage {
				if resp.Items[i].Name != want {
					t.Errorf("item[%d]: want %q, got %q", i, want, resp.Items[i].Name)
				}
			}
		})
	}
}

func callList(t *testing.T, base, filter, sort string, limit, offset uint64) *usersv1.ListUsersResponse {
	t.Helper()
	q := url.Values{}
	if filter != "" {
		q.Set("query.filter", filter)
	}
	if sort != "" {
		q.Set("query.sort", sort)
	}
	if limit > 0 {
		q.Set("query.limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		q.Set("query.offset", strconv.FormatUint(offset, 10))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/users?"+q.Encode(), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", httpResp.StatusCode, body)
	}

	// grpc-gateway's default marshaler emits camelCase JSON keys for proto
	// fields. Decode into a dedicated struct rather than the proto type so
	// we don't depend on protojson's tag interpretation.
	var out struct {
		Items []struct {
			ID     int64  `json:"id,string"`
			Name   string `json:"name"`
			Email  string `json:"email"`
			Status string `json:"status"`
		} `json:"items"`
		Total uint64 `json:"total,string"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", body, err)
	}
	resp := &usersv1.ListUsersResponse{Total: out.Total}
	for _, it := range out.Items {
		resp.Items = append(resp.Items, &usersv1.User{
			Id: it.ID, Name: it.Name, Email: it.Email, Status: it.Status,
		})
	}
	return resp
}

