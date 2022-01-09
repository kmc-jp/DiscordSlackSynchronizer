package discord_webhook

import "strings"

func FindContentType(FileName string) string {
	var sep = strings.Split(FileName, ".")

	switch strings.ToLower(sep[len(sep)-1]) {
	default:
		return "application/octet-stream"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	}
}
