// Package spisql provides primitives for parsing list-endpoint queries
// (filter / sort / pagination) and translating them into database-specific
// query languages.
//
// The root package exposes types only — Filter, Sort, Pagination, Query —
// kept zero-dep so it can be safely imported anywhere. Parsing lives in
// package parser; database backends and HTTP/gRPC transports are submodules
// under adapter/ and transport/ respectively.
//
// Wire format
//
// A list request carries four query keys:
//
//	$filter = name eq vasya,id in (1,2,3),status ne deleted
//	$sort   = name ASC,created_at DESC
//	$limit  = 25
//	$offset = 50
//
// Operators: eq, ne, in, nin, gt, gte, gtedate, lt, lte, substringof,
// startswith, endswith.
package spisql
