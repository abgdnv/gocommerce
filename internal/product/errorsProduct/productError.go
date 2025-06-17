// Package errorsProduct provides custom error types for product-related operations.
package errorsProduct

import "errors"

var ErrProductNotFound = errors.New("product not found")
var ErrCantCreateProduct = errors.New("can't create product")
