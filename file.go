package slogw

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// TimeFormatUnix defines a time format that makes time fields to be
	// serialized as Unix timestamp integers.
	TimeFormatUnix = "\x01"

	// TimeFormatUnixMs defines a time format that makes time fields to be
	// serialized as Unix timestamp integers in milliseconds.
	TimeFormatUnixMs = "\x02"
)

var (
	pid = os.Getpid()
)

// FileWriter is a Writer that writes to the specified filename.
//
// Backups use the log file name given to FileWriter, in the form
// `name.timestamp.ext` where name is the filename without the extension,
// timestamp is the time at which the log was rotated formatted with the
// time.Time format of `2006-01-02T15-04-05` and the extension is the
// original extension.  For example, if your FileWriter.Filename is
// `/var/log/foo/server.log`, a backup created at 6:30pm on Nov 11 2016 would
// use the filename `/var/log/foo/server.2016-11-04T18-30-00.log`
//
// # Cleaning Up Old Log Files
//
// Whenever a new logfile gets created, old log files may be deleted.  The most
// recent files according to filesystem modified time will be retained, up to a
// number equal to MaxBackups (or all of them if MaxBackups is 0). Note that the
// time encoded in the timestamp is the rotation time, which may differ from the
// last time that file was written to.
type FileWriter struct {
	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.
	Filename string

	// MaxSize is the maximum size in bytes of the log file before it gets rotated.
	MaxSize int64

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files
	MaxBackups int

	// make align check happy
	mu   sync.Mutex
	size int64
	file *os.File

	// FileMode represents the file's mode and permission bits.  The default
	// mode is 0644
	FileMode os.FileMode

	// TimeFormat specifies the time format of filename, uses `2006-01-02T15-04-05` as default format.
	// If set with `TimeFormatUnix`, `TimeFormatUnixMs`, times are formatted as UNIX timestamp.
	TimeFormat string

	// LocalTime determines if the time used for formatting the timestamps in
	// log files is the computer's local time.  The default is to use UTC time.
	LocalTime bool

	// ProcessID determines if the pid used for formatting in log files.
	ProcessID bool

	// EnsureFolder ensures the file directory creation before writing.
	EnsureFolder bool

	// Header specifies an optional header function of log file after rotation,
	Header func(fileInfo os.FileInfo) []byte

	// Cleaner specifies an optional cleanup function of log backups after rotation,
	// if not set, the default behavior is to delete more than MaxBackups log files.
	Cleaner func(fileName string, maxBackups int, matches []os.FileInfo)
}

// Write implements io.Writer.  If write would cause the log file to be larger
// than MaxSize, the file is closed, rotate to include a timestamp of the
// current time, and update symlink with log name file to the new file.
func (w *FileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	n, err = w.write(p)
	w.mu.Unlock()
	return
}

func (w *FileWriter) write(p []byte) (n int, err error) {
	if w.file == nil {
		if w.Filename == "" {
			n, err = os.Stderr.Write(p)
			return
		}
		if w.EnsureFolder {
			err = os.MkdirAll(filepath.Dir(w.Filename), 0755)
			if err != nil {
				return
			}
		}
		err = w.create()
		if err != nil {
			return
		}
	}

	n, err = w.file.Write(p)
	if err != nil {
		return
	}

	w.size += int64(n)
	if w.MaxSize > 0 && w.size > w.MaxSize && w.Filename != "" {
		err = w.rotate()
	}

	return
}

// Close implements io.Closer, and closes the current logfile.
func (w *FileWriter) Close() (err error) {
	w.mu.Lock()
	if w.file != nil {
		err = w.file.Close()
		w.file = nil
		w.size = 0
	}
	w.mu.Unlock()
	return
}

// Rotate causes Logger to close the existing log file and immediately create a
// new one.  This is a helper function for applications that want to initiate
// rotations outside the normal rotation rules, such as in response to SIGHUP.
// After rotating, this initiates compression and removal of old log
// files according to the configuration.
func (w *FileWriter) Rotate() (err error) {
	w.mu.Lock()
	err = w.rotate()
	w.mu.Unlock()
	return
}

func (w *FileWriter) rotate() (err error) {
	var file *os.File
	file, err = os.OpenFile(w.fileArgs(time.Now()))
	if err != nil {
		return err
	}
	if w.file != nil {
		w.file.Close()
	}
	w.file = file
	w.size = 0

	if w.Header != nil {
		st, err := file.Stat()
		if err != nil {
			return err
		}
		if b := w.Header(st); b != nil {
			n, err := w.file.Write(b)
			w.size += int64(n)
			if err != nil {
				return nil
			}
		}
	}

	go func(newName string) {
		os.Remove(w.Filename)
		if !w.ProcessID {
			_ = os.Symlink(filepath.Base(newName), w.Filename)
		}

		uid, _ := strconv.Atoi(os.Getenv("SUDO_UID"))
		gid, _ := strconv.Atoi(os.Getenv("SUDO_GID"))
		if uid != 0 && gid != 0 && os.Geteuid() == 0 {
			_ = os.Lchown(w.Filename, uid, gid)
			_ = os.Chown(newName, uid, gid)
		}

		dir := filepath.Dir(w.Filename)
		dirfile, err := os.Open(dir)
		if err != nil {
			return
		}
		infos, err := dirfile.Readdir(-1)
		dirfile.Close()
		if err != nil {
			return
		}

		base, ext := filepath.Base(w.Filename), filepath.Ext(w.Filename)
		prefix, extgz := base[:len(base)-len(ext)]+".", ext+".gz"
		exclude := prefix + "error" + ext

		matches := make([]os.FileInfo, 0)
		for _, info := range infos {
			name := info.Name()
			if name != base && name != exclude &&
				strings.HasPrefix(name, prefix) &&
				(strings.HasSuffix(name, ext) || strings.HasSuffix(name, extgz)) {
				matches = append(matches, info)
			}
		}
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].ModTime().Unix() < matches[j].ModTime().Unix()
		})

		if w.Cleaner != nil {
			w.Cleaner(w.Filename, w.MaxBackups, matches)
		} else {
			for i := 0; i < len(matches)-w.MaxBackups-1; i++ {
				_ = os.Remove(filepath.Join(dir, matches[i].Name()))
			}
		}
	}(w.file.Name())

	return
}

func (w *FileWriter) create() (err error) {
	w.file, err = os.OpenFile(w.fileArgs(time.Now()))
	if err != nil {
		return err
	}
	w.size = 0
	st, err := w.file.Stat()
	if err == nil {
		w.size = st.Size()
	}

	if w.size == 0 && w.Header != nil {
		if b := w.Header(st); b != nil {
			n, err := w.file.Write(b)
			w.size += int64(n)
			if err != nil {
				return err
			}
		}
	}

	err = os.Remove(w.Filename)
	if !w.ProcessID {
		_ = os.Symlink(filepath.Base(w.file.Name()), w.Filename)
	}

	return
}

// fileArgs returns a new filename, flag, perm based on the original name and the given time.
func (w *FileWriter) fileArgs(now time.Time) (filename string, flag int, perm os.FileMode) {
	if !w.LocalTime {
		now = now.UTC()
	}

	// filename
	ext := filepath.Ext(w.Filename)
	prefix := w.Filename[0 : len(w.Filename)-len(ext)]
	switch w.TimeFormat {
	case "":
		filename = prefix + now.Format(".2006-01-02T15-04-05")
	case TimeFormatUnix:
		filename = prefix + "." + strconv.FormatInt(now.Unix(), 10)
	case TimeFormatUnixMs:
		filename = prefix + "." + strconv.FormatInt(now.UnixNano()/1000000, 10)
	default:
		filename = prefix + "." + now.Format(w.TimeFormat)
	}
	if w.ProcessID {
		filename += "." + strconv.Itoa(pid) + ext
	} else {
		filename += ext
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
