# session-web — WhatsApp Session Generator (website)

A tiny, **standalone** website: enter a WhatsApp number, get a pair code,
link the device, and it shows you a **short** session string right in the
browser. It does not talk to Mongo, does not write files, does not log the
session anywhere — once you copy it and close the tab, it's gone from the
server too (each result can only be fetched once, and unclaimed pairings
are dropped after 10 minutes).

This is separate from the main `PbxGb` bot repo on purpose — deploy it as
its own small Heroku app (or run it locally) whenever you need a fresh
session, then paste the string into the bot's `SESSION` config var (or
straight into its `session-gen` tool, which will also push it to Mongo for
you).

## Run locally

```
go mod tidy
go run .
```

Open http://localhost:8000, enter your number (digits only, with country
code, e.g. `919646191598`), click **Get Pair Code**, and link it from
WhatsApp → Linked Devices → Link with phone number. The page will then show
your session string.

## Deploy to Heroku

```
heroku create your-session-tool
git init && git add . && git commit -m "session-web"
heroku buildpacks:set heroku/go
git push heroku main
heroku ps:scale web=1
```

No config vars are required — this app has no database and stores nothing
between requests.

⚠️ Since this runs on the open internet if you deploy it, anyone who finds
the URL could request a pair code for **any** number they type in (though
only whoever controls that phone can actually approve the pairing). Don't
share the URL publicly, or add your own auth in front of it (e.g. a
Heroku-level basic-auth buildpack, or a shared secret query param check in
`handlePair`) if you're worried about that.

## How the short session works

This uses the same compact format as the `PbxGb` bot's `session-gen` tool:
only the fixed-size device identity (noise key, identity key, signed
prekey, registration ID, adv secret key, account signature, your JID) is
encoded — not the full growing SQLite session dump. That's why it stays
short (well under 1,000 characters) no matter what.
