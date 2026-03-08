#!/bin/sh
# test_install.sh — Runs inside the Docker container to verify install.sh works.
#
# This script:
# 1. Starts a local HTTP server to mock GitHub Releases
# 2. Patches install.sh to point at the mock server
# 3. Runs install.sh as a non-root user
# 4. Verifies the binary was installed correctly
set -e

echo "=== olLANma Install Script Test ==="

# 1. Start a local HTTP server to mock GitHub Releases
cd /test/mock-release
python3 -m http.server 9999 &
SERVER_PID=$!
sleep 1

# 2. Patch install.sh to point at our mock server instead of GitHub
cd /test
sed \
  -e 's|https://github.com/${REPO}/releases/latest/download|http://127.0.0.1:9999|g' \
  -e 's|https://github.com/henriquejaques/olLANma/releases/latest/download|http://127.0.0.1:9999|g' \
  install.sh.orig > install.sh
chmod +x install.sh

# 3. Run the install script as the test user
echo "--- Running install.sh ---"
su testuser -c "sh /test/install.sh"

# 4. Verify the binary was installed correctly
echo ""
echo "--- Verification ---"

# Check binary exists
if [ ! -f /usr/local/bin/olLANma ]; then
  echo "FAIL: /usr/local/bin/olLANma not found"
  kill $SERVER_PID 2>/dev/null
  exit 1
fi

# Check it's executable
if [ ! -x /usr/local/bin/olLANma ]; then
  echo "FAIL: /usr/local/bin/olLANma is not executable"
  kill $SERVER_PID 2>/dev/null
  exit 1
fi

# Check version output
VERSION_OUTPUT=$(olLANma --version)
echo "Version output: $VERSION_OUTPUT"

if echo "$VERSION_OUTPUT" | grep -q "install-test"; then
  echo "PASS: Version string matches expected build"
else
  echo "FAIL: Unexpected version output"
  kill $SERVER_PID 2>/dev/null
  exit 1
fi

# Check help works
olLANma --help > /dev/null 2>&1
echo "PASS: --help exits cleanly"

# Check skip-setup works (no config file, should NOT trigger setup wizard)
SKIP_OUTPUT=$(olLANma --skip-setup create 2>&1 || true)
if echo "$SKIP_OUTPUT" | grep -q "Welcome to olLANma"; then
  echo "FAIL: --skip-setup still triggered first-run wizard"
  kill $SERVER_PID 2>/dev/null
  exit 1
fi
if echo "$SKIP_OUTPUT" | grep -q "unknown command"; then
  echo "PASS: --skip-setup bypassed setup wizard and command reached runtime validation"
else
  echo "FAIL: --skip-setup behavior was not as expected"
  echo "Output was:"
  echo "$SKIP_OUTPUT"
  kill $SERVER_PID 2>/dev/null
  exit 1
fi

kill $SERVER_PID 2>/dev/null
echo ""
echo "=== All install script tests passed ==="
