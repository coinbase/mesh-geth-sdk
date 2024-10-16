package headers

import (
	"context"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
)

// TODO: filter headers by "interesting headers"
func ContextWithHeaders(r *http.Request) context.Context {
	ctx := r.Context()
	ctxWithHeaders := rpc.NewContextWithHeaders(ctx, r.Header)
	return ctxWithHeaders
}
