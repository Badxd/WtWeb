# Session Generator (Heroku, code-only)

This ZIP contains the standalone session-generator app only. No vendored WhatsMeow source.

## Deploy
Upload these files to the GitHub repository root (not inside a subfolder). Keep the Heroku Go buildpack and the Procfile. The buildpack downloads dependencies from go.mod/go.sum.

## Important
This keeps the existing pinned WhatsMeow version. It does NOT claim to update to the latest upstream WhatsMeow because a verified current module version is not available here.

The WebSocket disconnect observed in Heroku logs may still happen; additional connection/disconnect logging has been added to help identify the cause. No live Heroku or WhatsApp pairing test was performed.
