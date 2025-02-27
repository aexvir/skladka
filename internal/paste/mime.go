package paste

// https://developer.mozilla.org/en-US/docs/Web/HTTP/MIME_types/Common_types
const (
	MimeUknown = "unknown"
	MimeAAC    = "audio/aac"
	MimeAbw    = "application/x-abiword"
	MimeApng   = "image/apng"
	MimeArc    = "application/x-freearc"
	MimeAvif   = "image/avif"
	MimeAvi    = "video/x-msvideo"
	MimeAzw    = "application/vnd.amazon.ebook"
	MimeBin    = "application/octet-stream"
	MimeBmp    = "image/bmp"
	MimeBz     = "application/x-bzip"
	MimeBz2    = "application/x-bzip2"
	MimeCda    = "application/x-cdf"
	MimeCsh    = "application/x-csh"
	MimeCss    = "text/css"
	MimeCsv    = "text/csv"
	MimeDoc    = "application/msword"
	MimeDocx   = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimeEot    = "application/vnd.ms-fontobject"
	MimeEpub   = "application/epub+zip"
	MimeGz     = "application/gzip"
	MimeGif    = "image/gif"
	MimeHtml   = "text/html"
	MimeIco    = "image/vnd.microsoft.icon"
	MimeIcs    = "text/calendar"
	MimeJar    = "application/java-archive"
	MimeJpeg   = "image/jpeg"
	MimeJs     = "text/javascript"
	MimeJson   = "application/json"
	MimeJsonld = "application/ld+json"
	MimeMidi   = "audio/midi"
	// MimeMjs    = "text/javascript"
	MimeMp3  = "audio/mpeg"
	MimeMp4  = "video/mp4"
	MimeMpeg = "video/mpeg"
	MimeMpkg = "application/vnd.apple.installer+xml"
	MimeOdp  = "application/vnd.oasis.opendocument.presentation"
	MimeOds  = "application/vnd.oasis.opendocument.spreadsheet"
	MimeOdt  = "application/vnd.oasis.opendocument.text"
	// MimeOga   = "audio/ogg"
	MimeOgv   = "video/ogg"
	MimeOgx   = "application/ogg"
	MimeOpus  = "audio/ogg"
	MimeOtf   = "font/otf"
	MimePng   = "image/png"
	MimePdf   = "application/pdf"
	MimePhp   = "application/x-httpd-php"
	MimePpt   = "application/vnd.ms-powerpoint"
	MimePptx  = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	MimeRar   = "application/vnd.rar"
	MimeRtf   = "application/rtf"
	MimeSh    = "application/x-sh"
	MimeSvg   = "image/svg+xml"
	MimeTar   = "application/x-tar"
	MimeTiff  = "image/tiff"
	MimeTs    = "video/mp2t"
	MimeTtf   = "font/ttf"
	MimeTxt   = "text/plain"
	MimeVsd   = "application/vnd.visio"
	MimeWav   = "audio/wav"
	MimeWeba  = "audio/webm"
	MimeWebm  = "video/webm"
	MimeWebp  = "image/webp"
	MimeWoff  = "font/woff"
	MimeWoff2 = "font/woff2"
	MimeXhtml = "application/xhtml+xml"
	MimeXls   = "application/vnd.ms-excel"
	MimeXlsx  = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	MimeXml   = "application/xml"
	MimeXul   = "application/vnd.mozilla.xul+xml"
	MimeZip   = "application/zip"
	Mime3gp   = "video/3gpp"
	Mime3g2   = "video/3gpp2"
	Mime7z    = "application/x-7z-compressed"
)

var MimetypeFileExtension = map[string]string{
	MimeUknown: ".txt",
	MimeAAC:    ".aac",
	MimeAbw:    ".abw",
	MimeApng:   ".apng",
	MimeArc:    ".arc",
	MimeAvif:   ".avif",
	MimeAvi:    ".avi",
	MimeAzw:    ".azw",
	MimeBin:    ".bin",
	MimeBmp:    ".bmp",
	MimeBz:     ".bz",
	MimeBz2:    ".bz2",
	MimeCda:    ".cda",
	MimeCsh:    ".csh",
	MimeCss:    ".css",
	MimeCsv:    ".csv",
	MimeDoc:    ".doc",
	MimeDocx:   ".docx",
	MimeEot:    ".eot",
	MimeEpub:   ".epub",
	MimeGz:     ".gz",
	MimeGif:    ".gif",
	MimeHtml:   ".html",
	MimeIco:    ".ico",
	MimeIcs:    ".ics",
	MimeJar:    ".jar",
	MimeJpeg:   ".jpg",
	MimeJs:     ".js",
	MimeJson:   ".json",
	MimeJsonld: ".jsonld",
	MimeMidi:   ".midi",
	// MimeMjs:    ".mjs",
	MimeMp3:  ".mp3",
	MimeMp4:  ".mp4",
	MimeMpeg: ".mpeg",
	MimeMpkg: ".mpkg",
	MimeOdp:  ".odp",
	MimeOds:  ".ods",
	MimeOdt:  ".odt",
	// MimeOga:   ".oga",
	MimeOgv:   ".ogv",
	MimeOgx:   ".ogx",
	MimeOpus:  ".opus",
	MimeOtf:   ".otf",
	MimePng:   ".png",
	MimePdf:   ".pdf",
	MimePhp:   ".php",
	MimePpt:   ".ppt",
	MimePptx:  ".pptx",
	MimeRar:   ".rar",
	MimeRtf:   ".rtf",
	MimeSh:    ".sh",
	MimeSvg:   ".svg",
	MimeTar:   ".tar",
	MimeTiff:  ".tiff",
	MimeTs:    ".ts",
	MimeTtf:   ".ttf",
	MimeTxt:   ".txt",
	MimeVsd:   ".vsd",
	MimeWav:   ".wav",
	MimeWeba:  ".weba",
	MimeWebm:  ".webm",
	MimeWebp:  ".webp",
	MimeWoff:  ".woff",
	MimeWoff2: ".woff2",
	MimeXhtml: ".xhtml",
	MimeXls:   ".xls",
	MimeXlsx:  ".xlsx",
	MimeXml:   ".xml",
	MimeXul:   ".xul",
	MimeZip:   ".zip",
	Mime3gp:   ".3gp",
	Mime3g2:   ".3g2",
	Mime7z:    ".7z",
}
