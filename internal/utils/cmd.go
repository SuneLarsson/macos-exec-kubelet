package utils

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// BuildExportEnvCommand returns a shell command that exports the given environment variable.
// The command is formatted as "export NAME=VALUE" where VALUE is escaped using heredoc syntax if it contains newlines.
func BuildExportEnvCommand(env corev1.EnvVar) string {
	var value string
	switch {
	case strings.Count(env.Value, "\n") == 0:
		value = strconv.Quote(env.Value)
	default:
		// support multineline env variables
		value = "$(cat <<'ESCAPE_EOF'\n" + env.Value + "\nESCAPE_EOF\n)"
	}
	return fmt.Sprintf("export %s=%s\n", env.Name, value)
}

// BuildExecCommandString returns a shell command that executes the given command in a shell.
// The command is formatted as "sh -c $'COMMAND'" where COMMAND is the given command string.
// If the command has arguments, they are appended to the command string.
func BuildExecCommandString(cmd []string, env []corev1.EnvVar) (string, error) {
	if len(cmd) < 3 || cmd[1] != "-c" {
		return "", fmt.Errorf("command is not a shell exec command")
	}

	cmdStr := ""
	for _, e := range env {
		cmdStr += BuildExportEnvCommand(e)
	}

	// If the -c option is present, then commands are read from string.
	cmdStr += cmd[0] + " " + cmd[1] // e.g. "sh -c"
	cmdStr += fmt.Sprintf(" $'%s'", cmd[2])

	// If there are arguments after the string, they are assigned to the positional parameters, starting with $0.
	for i := 3; i < len(cmd); i++ {
		// add arguments to sh -c command if any
		cmdStr += " " + strconv.Quote(cmd[i])
	}

	return cmdStr, nil
}
