package main

type ProgressReporter struct {
	AppendLog   func(string)
	RewriteLog  func(string)
	SetProgress func(status string, value float64)
	Clear       func()
}

func (p ProgressReporter) appendLog(text string) {
	if p.AppendLog != nil {
		p.AppendLog(text)
	}
}

func (p ProgressReporter) rewriteLog(text string) {
	if p.RewriteLog != nil {
		p.RewriteLog(text)
	} else {
		p.appendLog(text)
	}
}

func (p ProgressReporter) setProgress(status string, value float64) {
	if p.SetProgress != nil {
		p.SetProgress(status, value)
	}
}

func (p ProgressReporter) clear() {
	if p.Clear != nil {
		p.Clear()
	}
}
