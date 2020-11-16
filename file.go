package log

import (
	"crypto/md5"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FileWriter is an Writer that writes to the specified filename.
//
// Backups use the log file name given to FileWriter, in the form
// `name.timestamp.ext` where name is the filename without the extension,
// timestamp is the time at which the log was rotated formatted with the
// time.Time format of `2006-01-02T15-04-05` and the extension is the
// original extension.  For example, if your FileWriter.Filename is
// `/var/log/foo/server.log`, a backup created at 6:30pm on Nov 11 2016 would
// use the filename `/var/log/foo/server.2016-11-04T18-30-00.log`
//
// Cleaning Up Old Log Files
//
// Whenever a new logfile gets created, old log files may be deleted.  The most
// recent files according to the encoded timestamp will be retained, up to a
// number equal to MaxBackups (or all of them if MaxBackups is 0).  Any files
// with an encoded timestamp older than MaxAge days are deleted, regardless of
// MaxBackups.  Note that the time encoded in the timestamp is the rotation
// time, which may differ from the last time that file was written to.
type FileWriter struct {
	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.
	Filename string

	// MaxSize is the maximum size in bytes of the log file before it gets rotated.
	MaxSize int64

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files
	MaxBackups int

	// make aligncheck happy
	mu   sync.Mutex
	size int64
	file *os.File

	// FileMode represents the file's mode and permission bits.  The default
	// mode is 0644
	FileMode os.FileMode

	// EnsureDir determines if the time used for formatting the timestamps in
	// log files is the computer's local time.  The default is to use UTC time.
	EnsureDir bool

	// LocalTime determines if the time used for formatting the timestamps in
	// log files is the computer's local time.  The default is to use UTC time.
	LocalTime bool

	// HostName determines if the hostname used for formatting in log files.
	HostName bool

	// ProcessID determines if the pid used for formatting in log files.
	ProcessID bool
}

// WriteEntry implements Writer.  If a write would cause the log file to be larger
// than MaxSize, the file is closed, renamed to include a timestamp of the
// current time, and a new log file is created using the original log file name.
// If the length of the write is greater than MaxSize, an error is returned.
func (w *FileWriter) WriteEntry(e *Entry) (n int, err error) {
	return w.Write(e.buf)
}

// Write implements io.Writer.  If a write would cause the log file to be larger
// than MaxSize, the file is closed, renamed to include a timestamp of the
// current time, and a new log file is created using the original log file name.
// If the length of the write is greater than MaxSize, an error is returned.
func (w *FileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()

	if w.file == nil {
		if w.Filename == "" {
			n, err = os.Stderr.Write(p)
			w.mu.Unlock()
			return
		}
		if w.EnsureDir {
			err = os.MkdirAll(filepath.Dir(w.Filename), 0755)
			if err != nil {
				w.mu.Unlock()
				return
			}
		}
		err = w.create()
		if err != nil {
			w.mu.Unlock()
			return
		}
	}

	n, err = w.file.Write(p)
	if err != nil {
		w.mu.Unlock()
		return
	}

	w.size += int64(n)
	if w.MaxSize > 0 && w.size > w.MaxSize && w.Filename != "" {
		err = w.rotate()
	}

	w.mu.Unlock()
	return
}

// Close implements io.Closer, and closes the current logfile.
func (w *FileWriter) Close() (err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		err = w.file.Close()
		w.file = nil
		w.size = 0
	}
	return
}

// Rotate causes Logger to close the existing log file and immediately create a
// new one.  This is a helper function for applications that want to initiate
// rotations outside of the normal rotation rules, such as in response to
// SIGHUP.  After rotating, this initiates compression and removal of old log
// files according to the configuration.
func (w *FileWriter) Rotate() (err error) {
	w.mu.Lock()
	err = w.rotate()
	w.mu.Unlock()
	return
}

func (w *FileWriter) rotate() (err error) {
	oldfile := w.file

	w.file, err = os.OpenFile(w.fileinfo(timeNow()))
	if err != nil {
		return err
	}
	w.size = 0

	go func(oldfile *os.File, newname, filename string, backups int, processID bool) {
		if oldfile != nil {
			oldfile.Close()
		}

		os.Remove(filename)
		if !processID {
			os.Symlink(filepath.Base(newname), filename)
		}

		uid, _ := strconv.Atoi(os.Getenv("SUDO_UID"))
		gid, _ := strconv.Atoi(os.Getenv("SUDO_GID"))
		if uid != 0 && gid != 0 && os.Geteuid() == 0 {
			os.Lchown(filename, uid, gid)
			os.Chown(newname, uid, gid)
		}

		ext := filepath.Ext(filename)
		pattern := filename[0:len(filename)-len(ext)] + ".20*" + ext
		if names, _ := filepath.Glob(pattern); len(names) > 0 {
			sort.Strings(names)
			for i := 0; i < len(names)-backups-1; i++ {
				os.Remove(names[i])
			}
		}
	}(oldfile, w.file.Name(), w.Filename, w.MaxBackups, w.ProcessID)

	return
}

func (w *FileWriter) create() (err error) {
	w.file, err = os.OpenFile(w.fileinfo(timeNow()))
	if err != nil {
		return err
	}
	w.size = 0

	os.Remove(w.Filename)
	if !w.ProcessID {
		os.Symlink(filepath.Base(w.file.Name()), w.Filename)
	}

	return
}

// fileinfo returns a new filename, flag, perm based on the original name and the given time.
func (w *FileWriter) fileinfo(now time.Time) (filename string, flag int, perm os.FileMode) {
	if !w.LocalTime {
		now = now.UTC()
	}

	// filename
	ext := filepath.Ext(w.Filename)
	prefix := w.Filename[0 : len(w.Filename)-len(ext)]
	filename = prefix + now.Format(".2006-01-02T15-04-05")
	if w.HostName {
		if w.ProcessID {
			filename += "." + hostname + "-" + strconv.Itoa(pid) + ext
		} else {
			filename += "." + hostname + ext
		}
	} else {
		if w.ProcessID {
			filename += "." + strconv.Itoa(pid) + ext
		} else {
			filename += ext
		}
	}

	// flag
	flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY

	// perm
	perm = w.FileMode
	if perm == 0 {
		perm = 0644
	}

	return
}

var hostname, machine = func() (string, [16]byte) {
	// host
	host, err := os.Hostname()
	if err != nil || strings.HasPrefix(host, "localhost") {
		host = "localhost-" + strconv.FormatInt(int64(Fastrandn(1000000)), 10)
	}
	// seed files
	var files []string
	switch runtime.GOOS {
	case "linux":
		files = []string{"/etc/machine-id", "/proc/self/cpuset"}
	case "freebsd":
		files = []string{"/etc/hostid"}
	}
	// append seed to hostname
	data := []byte(host)
	for _, file := range files {
		if b, err := ioutil.ReadFile(file); err == nil {
			data = append(data, b...)
		}
	}
	// md5 digest
	hex := md5.Sum(data)

	return host, hex
}()

var pid = os.Getpid()

var _ Writer = (*FileWriter)(nil)
var _ io.Writer = (*FileWriter)(nil)
