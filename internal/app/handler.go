package app

import (
	"io"
	"log/slog"
)

type handler interface {
	Handler(io.Writer, *slog.HandlerOptions) slog.Handler
}

type handlerType string

const (
	handlerTypeJSON handlerType = "json"
	handlerTypeText handlerType = "text"
)

func textHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.NewTextHandler(w, opts)
}

func jsonHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.NewJSONHandler(w, opts)
}

func (h handlerType) Handler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return handlers[h](w, opts)
}

type handlerFunc func(io.Writer, *slog.HandlerOptions) slog.Handler

var handlers = map[handlerType]handlerFunc{
	handlerTypeText: textHandler,
	handlerTypeJSON: jsonHandler,
}

// GetHandlerType returns the appropriate handler type based on format string.
func GetHandlerType(format LogFormat) handlerType {
	switch format {
	case "json":
		return handlerTypeJSON
	case "text":
		return handlerTypeText
	default:
		return handlerTypeText
	}
}

var _ handler = handlerType("")
