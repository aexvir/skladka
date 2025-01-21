package attributes

import "go.opentelemetry.io/otel/attribute"

const (
	KeyPasteReference = "paste.ref"
	KeyPasteTitle     = "paste.title"
	KeyPasteSyntax    = "paste.syntax"
	KeyPasteTags      = "paste.tags"
	KeyPasteSize      = "paste.size"
)

func PasteAttributeSet(ref, title, syntax string, tags []string) []attribute.KeyValue {
	return []attribute.KeyValue{
		PasteReference(ref),
		PasteTitle(title),
		PasteSyntax(syntax),
		PasteTags(tags),
	}
}

func PasteReference(ref string) attribute.KeyValue {
	return String(KeyPasteReference, ref)
}

func PasteTitle(title string) attribute.KeyValue {
	return String(KeyPasteReference, title)
}

func PasteSyntax(syntax string) attribute.KeyValue {
	return String(KeyPasteSyntax, syntax)
}

func PasteTags(tags []string) attribute.KeyValue {
	return StringSlice(KeyPasteTags, tags)
}
