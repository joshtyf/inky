package core

type CoreErrorSeverity int

const (
	CoreErrorSeverityError CoreErrorSeverity = iota
	CoreErrorSeverityFatal
)

type CoreError interface {
	error
	GetSeverity() CoreErrorSeverity
}
