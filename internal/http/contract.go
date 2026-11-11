//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=httpserver_test

package httpserver

import "context"

type pinger interface{ Ping(context.Context) error }
