package log

import (
	"testing"
)

func TestMultiFileWriter(t *testing.T) {
	w := &MultiFileWriter{
		Writes: make(map[string]Writer),
	}
	w.Writes["heartbeat"] = &FileWriter{Filename: "file-heartbeat.log"}
	w.Writes["message"] = &FileWriter{Filename: "file-message.log"}

	for _, level := range []string{"trace", "debug", "info", "warning", "error", "fatal", "panic", "hahaha"} {
		_, err := loggerPrintf(w, "heartbeat", ParseLevel(level), `{"ts":1234567890,"level":"%s","caller":"test.go:42","error":"i am test heartbeat","foo":"bar","n":42,"message":"hello json mutli writer"}`+"\n", level)
		if err != nil {
			t.Errorf("test json mutli writer error: %+v", err)
		}
		_, err = loggerPrintf(w, "message", ParseLevel(level), `{"time":"2019-07-10T05:35:54.277Z","level":"%s","caller":"test.go:42","error":"i am test message","foo":"bar","n":42,"message":"hello json mutli writer"}`+"\n", level)
		if err != nil {
			t.Errorf("test json mutli writer error: %+v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Errorf("test close mutli writer error: %+v", err)
	}
}
