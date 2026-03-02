#!/bin/bash
# create-runner-user.sh
#
# One-time setup script to create the 'vk-runner' system account on macOS.
# Run as an administrator: sudo bash scripts/create-runner-user.sh
#
# The account has:
#   - No login shell (/usr/bin/false)
#   - Home directory set to /var/empty (read-only, shared system dir)
#   - No password (cannot log in interactively)
#   - Full access to system frameworks (CoreML / ANE / Metal) as a normal user

set -euo pipefail

RUNNER_USER="vk-runner"
RUNNER_UID=599   # Change if UID 599 is already taken (check with: dscl . -list /Users UniqueID)
RUNNER_GID=20    # 20 = 'staff', a sensible default primary group

if dscl . -read /Users/"$RUNNER_USER" &>/dev/null; then
  echo "User '$RUNNER_USER' already exists. Nothing to do."
  exit 0
fi

echo "Creating system user '$RUNNER_USER' (UID=$RUNNER_UID)..."

dscl . -create /Users/"$RUNNER_USER"
dscl . -create /Users/"$RUNNER_USER" UserShell /usr/bin/false
dscl . -create /Users/"$RUNNER_USER" RealName "VK Job Runner"
dscl . -create /Users/"$RUNNER_USER" UniqueID "$RUNNER_UID"
dscl . -create /Users/"$RUNNER_USER" PrimaryGroupID "$RUNNER_GID"
dscl . -create /Users/"$RUNNER_USER" NFSHomeDirectory /var/empty

# Prevent the account from appearing on the login screen
defaults write /Library/Preferences/com.apple.loginwindow HiddenUsersList -array-add "$RUNNER_USER" 2>/dev/null || true

echo "Done. Verify with: id $RUNNER_USER"
id "$RUNNER_USER"
