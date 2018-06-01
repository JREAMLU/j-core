package ext

import "bytes"

// StringSplice string splice
func StringSplice(content ...string) string {
	var str bytes.Buffer
	for _, cnt := range content {
		str.WriteString(cnt)
	}

	return str.String()
}
