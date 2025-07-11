// Package errors provides custom error types for product-related operations.
package errors

import "errors"

var ErrCreateOrder = errors.New("failed to create order")
var ErrCreateOrderItem = errors.New("failed to create order item")

var ErrUpdateOrder = errors.New("failed to update order")
var ErrOptimisticLock = errors.New("optimistic lock error: the record has been modified by another transaction")

var ErrOrderNotFound = errors.New("order not found")
var ErrFailedToFindOrder = errors.New("failed to find order")
var ErrFailedToFindUserOrders = errors.New("failed to find user orders")

var ErrFailedToFindOrderItems = errors.New("failed to find order items")

var ErrTransactionBegin = errors.New("failed to begin transaction")
var ErrTransactionCommit = errors.New("failed to commit transaction")
var ErrTransactionRollback = errors.New("failed to rollback transaction")

var ErrAccessDenied = errors.New("access denied")
