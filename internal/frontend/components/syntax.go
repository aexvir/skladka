package components

// Syntax is a language supported by the editor's highlighter.
type Syntax struct {
	// Name is the highlights language id, stored with the paste.
	Name string `json:"name"`
	// Extension is used when downloading the paste as a file.
	Extension string `json:"extension"`
}

// Syntaxes lists the languages built into @pierre/highlights, plaintext first.
var Syntaxes = []Syntax{
	{"plaintext", "txt"},
	{"angular-html", "html"},
	{"asm", "s"},
	{"astro", "astro"},
	{"bash", "sh"},
	{"bat", "bat"},
	{"c", "c"},
	{"c3", "c3"},
	{"clojure", "clj"},
	{"cmake", "cmake"},
	{"cpp", "cpp"},
	{"csharp", "cs"},
	{"css", "css"},
	{"cuda", "cu"},
	{"dart", "dart"},
	{"diff", "diff"},
	{"dockerfile", "dockerfile"},
	{"elixir", "ex"},
	{"elm", "elm"},
	{"erlang", "erl"},
	{"fortran", "f90"},
	{"fortran-fixed-form", "f"},
	{"fsharp", "fs"},
	{"gdscript", "gd"},
	{"gdshader", "gdshader"},
	{"gleam", "gleam"},
	{"glsl", "glsl"},
	{"go", "go"},
	{"godot-resource", "tres"},
	{"graphql", "graphql"},
	{"groovy", "groovy"},
	{"haskell", "hs"},
	{"hcl", "hcl"},
	{"hlsl", "hlsl"},
	{"html", "html"},
	{"java", "java"},
	{"javascript", "js"},
	{"json", "json"},
	{"json5", "json5"},
	{"jsonl", "jsonl"},
	{"jsx", "jsx"},
	{"julia", "jl"},
	{"kotlin", "kt"},
	{"latex", "tex"},
	{"less", "less"},
	{"lisp", "lisp"},
	{"lua", "lua"},
	{"makefile", "mk"},
	{"markdown", "md"},
	{"matlab", "m"},
	{"mdx", "mdx"},
	{"nim", "nim"},
	{"nix", "nix"},
	{"objective-c", "m"},
	{"ocaml", "ml"},
	{"pascal", "pas"},
	{"perl", "pl"},
	{"php", "php"},
	{"powershell", "ps1"},
	{"protobuf", "proto"},
	{"python", "py"},
	{"r", "r"},
	{"ruby", "rb"},
	{"rust", "rs"},
	{"sass", "sass"},
	{"scala", "scala"},
	{"scss", "scss"},
	{"solidity", "sol"},
	{"sql", "sql"},
	{"svelte", "svelte"},
	{"swift", "swift"},
	{"toml", "toml"},
	{"tsrx", "tsrx"},
	{"tsx", "tsx"},
	{"typescript", "ts"},
	{"vue", "vue"},
	{"wasm", "wat"},
	{"wgsl", "wgsl"},
	{"xml", "xml"},
	{"yaml", "yaml"},
	{"zig", "zig"},
}

// SyntaxNames returns the names of all supported syntaxes.
func SyntaxNames() []string {
	names := make([]string, len(Syntaxes))
	for i, syntax := range Syntaxes {
		names[i] = syntax.Name
	}
	return names
}
