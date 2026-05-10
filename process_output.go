package main

import (
	"bufio"
	"io"
	"runtime"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func decodeProcessLine(line []byte) string {
	out := line
	if runtime.GOOS == "windows" && !utf8.Valid(line) {
		if dec, _, derr := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), line); derr == nil {
			out = dec
		} else if dec2, _, derr2 := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), line); derr2 == nil {
			out = dec2
		}
	}
	return strings.TrimSpace(strings.TrimRight(string(out), "\r\n"))
}

func readPipe(r io.Reader, appendLog func(string)) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			appendLog(decodeProcessLine(line))
		}
		if err != nil {
			if err != io.EOF {
				appendLog("读取子进程输出出错: " + err.Error())
			}
			break
		}
	}
}
