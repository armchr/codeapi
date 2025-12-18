package base

type FileHolder struct {
	FileURI  string
	lineStrs []string
}

func NewFileHolder(uri string, content string) *FileHolder {
	// split content into lines by splitting on '\n' and '\r\n'
	lineStrs := []string{}
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lineStrs = append(lineStrs, string(content[start:i]))
			start = i + 1
		} else if content[i] == '\r' {
			if i+1 < len(content) && content[i+1] == '\n' {
				lineStrs = append(lineStrs, string(content[start:i]))
				start = i + 2
				i++
			} else {
				lineStrs = append(lineStrs, string(content[start:i]))
				start = i + 1
			}
		}
	}
	if start < len(content) {
		lineStrs = append(lineStrs, string(content[start:]))
	}

	return &FileHolder{
		FileURI:  uri,
		lineStrs: lineStrs,
	}
}

func (fh *FileHolder) GetLine(line int) string {
	if line < 0 || line >= len(fh.lineStrs) {
		return ""
	}
	return fh.lineStrs[line]
}

func (fh *FileHolder) FindNameInLine(lspClient LSPClient, name string, line int) int {
	if line < 0 || line >= len(fh.lineStrs) {
		return -1
	}
	lineStr := fh.lineStrs[line]
	namePart := lspClient.SymbolPartToMatch(name)
	for i := 0; i <= len(lineStr)-len(namePart); i++ {
		if lineStr[i:i+len(namePart)] == namePart {
			return i
		}
	}
	return -1
}

func (fh *FileHolder) FindNameInNextLines(lspClient LSPClient, name string, startLine, n int) int {
	for i := startLine; i < startLine+n && i < len(fh.lineStrs); i++ {
		if idx := fh.FindNameInLine(lspClient, name, i); idx != -1 {
			return idx
		}
	}
	return -1
}
