package incomingHeader

import "errors"

var errNilHeadersPool = errors.New("nil headers pool provided")

var errNilTxPool = errors.New("nil tx pool provided")

var errInvalidHeaderType = errors.New("incoming header is not of type HeaderV2")
