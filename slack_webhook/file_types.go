package slack_webhook

import "strings"

func FindFileType(FileName string) string {
	if FileName == "Dockerfile" {
		return "Dockerfile"
	}
	if !strings.Contains(FileName, ".") {
		return ""
	}

	var sep = strings.Split(FileName, ".")
	var Extension = sep[len(sep)-1]
	var extension = strings.ToLower(Extension)
	switch extension {
	default:
		return ""
	case "txt":
		return "text"
	case "ai", "apk", "bmp", "boxnote", "c", "cpp", "css", "csv", "clj", "cfm", "d", "dart", "diff", "doc", "docx", "puppet":
		return extension
	case "dotx", "eps", "epub", "fla", "flv", "gdoc", "gdraw", "gif", "go", "gpres", "groovy", "gsheet", "gzip", "html", "handlebars":
		return extension
	case "haxe", "indd", "java", "latex", "lisp", "lua", "m4a", "mhtml", "mkv", "mov", "mp3", "mp4", "mpg", "mumps", "nzb", "objc", "ocaml":
		return extension
	case "odg", "odi", "odp", "ods", "odt", "ogg", "ogv", "pages", "pascal", "pdf", "perl", "php", "pig", "png", "ppt", "pptx", "psd":
		return extension
	case "qtz", "r", "rtf", "sql", "sass", "scala", "scheme", "sketch", "smalltalk", "svg", "swf", "swift", "tar", "tiff", "tsv", "vb":
		return extension
	case "vcard", "velocity", "verilog", "wav", "webm", "wmv", "xls", "xlsx", "xlsb", "xlsm", "xltx", "xml", "yaml", "zip":
		return extension
	case "vbs":
		return "vbscript"
	case "sh":
		return "shell"
	case "rs":
		return "rust"
	case "rb":
		return "ruby"
	case "py":
		return "python"
	case "ps1":
		return "powershell"
	case "mat":
		return "matlab"
	case "md":
		return "markdown"
	case "kt":
		return "kotlin"
	case "key":
		return "keynote"
	case "jpg", "jpeg":
		return "jpg"
	case "js", "json":
		return "javascript"
	case "hs":
		return "haskell"
	case "f":
		return "fortran"
	case "fsi":
		return "fsharp"
	case "scpt":
		return "applescript"
	case "erl":
		return "erlang"
	case "cs":
		return "csharp"
	case "coffee":
		return "coffeescript"
	}
}
