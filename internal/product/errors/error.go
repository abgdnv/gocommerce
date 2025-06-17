// Package errors provides custom error types for product-related operations.
package errors

import "errors"

var ErrProductNotFound = errors.New("product not found")
var ErrCantCreateProduct = errors.New("can't create product")
