package web

type contextKey string

const UserIDKey = contextKey("userID")
const requestIDKey = contextKey("requestID")
