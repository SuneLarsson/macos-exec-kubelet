package client

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

// resolvedCredential holds the UID/GID of the runner user, resolved once at startup.
type resolvedCredential struct {
	uid uint32
	gid uint32
}

// lookupRunnerCredential resolves a macOS username to its UID and GID.
// Returns nil, nil when username is empty (meaning: run as the current user).
// Called once at ProcessClient construction so a bad username is caught immediately.
func lookupRunnerCredential(username string) (*resolvedCredential, error) {
	if username == "" {
		return nil, nil
	}

	u, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("runner user %q not found: %w", username, err)
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid UID %q for user %q: %w", u.Uid, username, err)
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid GID %q for user %q: %w", u.Gid, username, err)
	}

	return &resolvedCredential{uid: uint32(uid), gid: uint32(gid)}, nil
}

// applyCredential sets SysProcAttr on cmd so it runs as the runner user.
// No-op when cred is nil (current user).
func applyCredential(cmd *exec.Cmd, cred *resolvedCredential) {
	if cred == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: cred.uid,
			Gid: cred.gid,
		},
	}
}

// chownLogDir changes ownership of the log directory to the runner user
// so the child process can write stdout/stderr into it.
func chownLogDir(path string, cred *resolvedCredential) error {
	if cred == nil {
		return nil
	}
	return os.Chown(path, int(cred.uid), int(cred.gid))
}
