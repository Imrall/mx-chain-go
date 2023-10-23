package dblookupext

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrNotFoundInStorage signals that an item was not found in storage
var ErrNotFoundInStorage = errors.New("not found in storage")

var errCannotCastToBlockBody = errors.New("cannot cast to block body")

var errNilESDTSuppliesHandler = errors.New("nil esdt supplies handler")

var errMiniblockHeaderNotFound = errors.New("miniblock header not found")

var errWrongTypeAssertion = errors.New("wrong type assertion")

func newErrCannotSaveEpochByHash(what string, hash []byte, originalErr error) error {
	return fmt.Errorf("cannot save epoch num for [%s] hash [%s]: %w", what, hex.EncodeToString(hash), originalErr)
}
