package common

import (
	"context"
	"net/http"
	"strconv"

	"github.com/romanpitatelev/wallets-service/internal/entity"
)

const (
	DefaultLimit = 25
)

func GetUserInfo(ctx context.Context) entity.UserInfo {
	val, _ := ctx.Value(entity.UserInfo{}).(entity.UserInfo)

	return val
}

func ParseGetRequest(r *http.Request) entity.GetWalletsRequest {
	queryParams := r.URL.Query()

	parameters := entity.GetWalletsRequest{
		Sorting: queryParams.Get("sorting"),
		Filter:  queryParams.Get("filter"),
	}

	var (
		limit  int64
		offset int64
	)

	if d := queryParams.Get("descending"); d != "" {
		parameters.Descending, _ = strconv.ParseBool(d)
	}

	if l := queryParams.Get("limit"); l != "" {
		if limit, _ = strconv.ParseInt(l, 0, 64); limit == 0 {
			limit = DefaultLimit
		}

		parameters.Limit = int(limit)
	} else {
		parameters.Limit = DefaultLimit
	}

	if o := queryParams.Get("offset"); o != "" {
		offset, _ = strconv.ParseInt(o, 0, 64)
		parameters.Offset = int(offset)
	}

	return parameters
}
