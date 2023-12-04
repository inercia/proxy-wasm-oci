package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

var (
	errSrcNotDir = errors.New("source is not a directory")
	errDstExist  = errors.New("destination already exists")
)

// AtomicWriteFile atomically (as atomic as os.Rename allows) writes a file to a
// disk.
func AtomicWriteFile(filename string, reader io.Reader, mode os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Split(filename))
	if err != nil {
		return err
	}
	tempName := tempFile.Name()

	if _, err := io.Copy(tempFile, reader); err != nil {
		tempFile.Close() // return value is ignored as we are already on error path
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tempName, mode); err != nil {
		return err
	}

	return RenameWithFallback(tempName, filename)
}

// RenameWithFallback attempts to rename a file or directory, but falls back to
// copying in the event of a cross-device link error. If the fallback copy
// succeeds, src is still removed, emulating normal rename behavior.
func RenameWithFallback(src, dst string) error {
	_, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("cannot stat %s: %w", src, err)
	}

	err = os.Rename(src, dst)
	if err == nil {
		return nil
	}

	return renameFallback(err, src, dst)
}

// renameFallback attempts to determine the appropriate fallback to failed rename
// operation depending on the resulting error.
func renameFallback(err error, src, dst string) error {
	// Rename may fail if src and dst are on different devices; fall back to
	// copy if we detect that case. syscall.EXDEV is the common name for the
	// cross device link error which has varying output text across different
	// operating systems.
	terr, ok := err.(*os.LinkError)
	if !ok {
		return err
	} else if terr.Err != syscall.EXDEV {
		return fmt.Errorf("link error: cannot rename %s to %s: %w", src, dst, terr)
	}

	return renameByCopy(src, dst)
}

// renameByCopy attempts to rename a file or directory by copying it to the
// destination and then removing the src thus emulating the rename behavior.
func renameByCopy(src, dst string) error {
	var cerr error
	if dir, _ := IsDir(src); dir {
		cerr = CopyDir(src, dst)
		if cerr != nil {
			cerr = fmt.Errorf("copying directory failed: %w", cerr)
		}
	} else {
		cerr = copyFile(src, dst)
		if cerr != nil {
			cerr = fmt.Errorf("copying file failed: %w", cerr)
		}
	}

	if cerr != nil {
		return fmt.Errorf("rename fallback failed: cannot rename %s to %s: %w", src, dst, cerr)
	}

	err := os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("cannot delete %s: %w", src, err)
	}

	return nil
}

// IsDir determines is the path given is a directory or not.
func IsDir(name string) (bool, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return false, err
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("%q is not a directory", name)
	}
	return true, nil
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
func CopyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	// We use os.Lstat() here to ensure we don't fall in a loop where a symlink
	// actually links to a one of its parent directories.
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errSrcNotDir
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		return errDstExist
	}

	if err = os.MkdirAll(dst, fi.Mode()); err != nil {
		return fmt.Errorf("cannot mkdir %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("cannot read directory %s: %w", dst, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err = CopyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("copying directory failed: %w", err)
			}
		} else {
			// This will include symlinks, which is what we want when
			// copying things.
			if err = copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copying file failed: %w", err)
			}
		}
	}

	return nil
}

// IsFilenameSymlink determines if the given path is a symbolic link.
func IsFilenameSymlink(path string) (bool, error) {
	l, err := os.Lstat(path)
	if err != nil {
		return false, err
	}

	return IsSymlink(l), nil
}

// IsSymlink is used to determine if the fileinfo is a symbolic link.
func IsSymlink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}

// copyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all its contents will be replaced by the contents
// of the source file. The file mode will be copied from the source.
func copyFile(src, dst string) (err error) {
	if sym, err := IsFilenameSymlink(src); err != nil {
		return fmt.Errorf("symlink check failed: %w", err)
	} else if sym {
		if err := cloneSymlink(src, dst); err != nil {
			if runtime.GOOS == "windows" {
				// If cloning the symlink fails on Windows because the user
				// does not have the required privileges, ignore the error and
				// fall back to copying the file contents.
				//
				// ERROR_PRIVILEGE_NOT_HELD is 1314 (0x522):
				// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681385(v=vs.85).aspx
				if lerr, ok := err.(*os.LinkError); ok && lerr.Err != syscall.Errno(1314) {
					return err
				}
			} else {
				return err
			}
		} else {
			return nil
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}

	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return
	}

	// Check for write errors on Close
	if err = out.Close(); err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}

	err = os.Chmod(dst, si.Mode())
	return
}

// cloneSymlink will create a new symlink that points to the resolved path of sl.
// If sl is a relative symlink, dst will also be a relative symlink.
func cloneSymlink(sl, dst string) error {
	resolved, err := os.Readlink(sl)
	if err != nil {
		return err
	}

	return os.Symlink(resolved, dst)
}

func IsFileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
