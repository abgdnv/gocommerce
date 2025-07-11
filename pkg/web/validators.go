package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

// ParamValidator is a function type that validates a parameter.
type ParamValidator func(valueToTest int64) bool

func newComparisonValidator(valueInClosure int64, compareFn func(argValue, closedValue int64) bool) ParamValidator {
	return func(argValue int64) bool {
		return compareFn(argValue, valueInClosure)
	}
}

// gte returns a ParamValidator that checks if the argument is greater than or equal to the value captured in the closure.
func gte(valToCompareAgainst int64) ParamValidator {
	return newComparisonValidator(valToCompareAgainst, func(argValue, closedValue int64) bool {
		return argValue >= closedValue
	})
}

// gt returns a ParamValidator that checks if the argument is greater than the value captured in the closure.
func gt(valToCompareAgainst int64) ParamValidator {
	return newComparisonValidator(valToCompareAgainst, func(argValue, closedValue int64) bool {
		return argValue > closedValue
	})
}

func ParseValidateGte(r *http.Request, w http.ResponseWriter, logger *slog.Logger, key string, value int64) (int32, bool) {
	return parseValidate(r, w, logger, key, gte(value))
}

func ParseValidateGt(r *http.Request, w http.ResponseWriter, logger *slog.Logger, key string, value int64) (int32, bool) {
	return parseValidate(r, w, logger, key, gt(value))
}

func parseValidate(r *http.Request, w http.ResponseWriter, logger *slog.Logger, key string, pValidator ParamValidator) (int32, bool) {
	value := r.URL.Query().Get(key)
	if value == "" {
		RespondError(w, logger, http.StatusBadRequest, fmt.Sprintf("%s url parameter is required", key))
		return 0, false // Return false if the parameter is not present
	}
	intValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil || !pValidator(intValue) {
		RespondError(w, logger, http.StatusBadRequest, fmt.Sprintf("Invalid %s number: %s", key, value))
		return 0, false
	}
	return int32(intValue), true
}
