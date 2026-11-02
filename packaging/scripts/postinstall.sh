#!/bin/sh
set -e
cat <<'EOF'

  vscale installed.

  Next (ask your admin for a join token):

    vscale setup
    # or:
    vscale join <TOKEN> --server https://your-server.example
    vscale up

  Docs: https://github.com/zpol/vscale

EOF
