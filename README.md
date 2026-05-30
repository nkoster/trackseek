# trackseek

trackseek is a small Shazam-like lab project in Go.

The idea is simple:

- index audio files into a SQLite database
- store audio fingerprints
- take a short sample
- try to find the right track back

Right now this is both:

- a local CLI tool
- a small HTTP server with a match endpoint

# What it does

- reads audio from `.wav` and `.mp3`
- extracts peaks from the spectrum
- builds fingerprint hashes from anchor-target peak pairs in a small forward target zone
- stores fingerprints in SQLite
- matches a sample against stored tracks
- serves static files from `./static`
- accepts uploaded audio samples over HTTP
- returns match results as SSE with JSON data

# Project goal

This project is made for a practical test setup.
Not for perfect production matching yet.

So the focus is:

- simple
- understandable
- fast enough
- easy to extend later

# Current database

The database is SQLite.

Main tables:

- `artists`
- `tracks`
- `fingerprints`

artists

- `id`
- `name`

tracks

- `id`
- `path`
- `title`
- `artist_id`

fingerprints

- `hash`
- `track_id`
- `time_ms`

# Build

```bash
go build
```

Or run directly:

```bash
go run .
```

# Configuration

## `.env`

At startup, trackseek tries to load a local `.env` file.

Right now this is used for the SQLite database path.

Without an override, the fallback database path is relative:

- `fingerprints.sqlite`
- this is resolved relative to the current working directory

Example:

```env
TRACKSEEK_DB_PATH=/home/niels/db/fingerprints.sqlite
```

If `.env` is missing, trackseek falls back to:

```text
fingerprints.sqlite
```

Using `.env` can make the database location explicit and stable.
That is useful when you want the DB file to stay in a fixed place,
regardless of how or from where you start the program.

At startup, trackseek prints which SQLite file it is using.

# Usage

## Index a track

Use `index` with title and artist:

```bash
./trackseek index --artist="Nortsch" --title="Time doesnt exist" ./song.mp3
```

This will:

- read the audio file
- create/find the artist
- create the track row
- store fingerprints

## Match a sample

Basic:

```bash
./trackseek match ./sample.mp3
```

With a minimum accepted score:

```bash
./trackseek match --min-score=400 ./sample.mp3
```

With early stop for clear matches:

```bash
./trackseek match --min-score=400 --threshold=1100 ./sample.mp3
```

## Start the HTTP server

Basic:

```bash
./trackseek serve
```

With a custom address:

```bash
./trackseek serve --addr :8081
```

With in-memory fingerprint preload:

```bash
./trackseek serve --preload
```

When the server starts:

- it serves files from `./static`
- `GET /` returns `static/index.html` when present
- asset files like `.js`, `.css`, images, and `/assets/...` are served directly
- unknown frontend routes fall back to `static/index.html`
- `POST /match` accepts an uploaded sample file
- with `--preload`, fingerprint hashes are loaded into an in-memory index at startup
- with `--preload`, `/match` uses the in-memory index instead of SQL fingerprint lookups

# Matching flags

Offsets are grouped in 100 ms buckets. This makes nearby hits count together.

## `--min-score`

This is the minimum score needed to accept a match.

`400` is a good starting point.

Example:

- if best score is `157`
- and `--min-score=400`
- then result is `no matching track found`

## `--threshold`

This is an early stop value.

If a candidate reaches this score during matching,
the matcher stops early and returns that result.

This is useful when you want to save time and resources.

`1100` is a good starting point.

If this happens, the output shows:

```text
[early stopped]
```

That means the match was accepted early.

# HTTP API

## Static files

The server uses the `static/` directory.

This is intended for static HTML now,
and later for a React bundle.

It can also work as a small SPA server:

- existing asset files are served directly
- unknown routes fall back to `index.html`

This is useful for React Router or other client-side routing.

Main route:

- `GET /`

## `POST /match`

This route accepts a multipart form upload.

Form fields:

- `sample`
- `minScore` optional
- `threshold` optional

The `sample` field should contain a `.wav` or `.mp3` file.

Example with `curl`:

```bash
curl -N \
  -F "sample=@./match-test.mp3" \
  -F "minScore=400" \
  -F "threshold=1100" \
  http://localhost:8080/match
```

If the server was started with `--preload`, this route matches against the preloaded in-memory fingerprint index.
Otherwise it uses the SQLite fingerprint table during the request.

## SSE response

The route returns:

```text
Content-Type: text/event-stream
```

It sends one SSE event named `match`.

Example successful response:

```text
event: match
data: {"matched":true,"trackId":17,"title":"Time doesnt exist","artist":"Nortsch","path":"./nortsch-time.mp3","score":289,"offsetMs":69474}
```

Example no-match response:

```text
event: match
data: {"matched":false}
```

Example error response:

```text
event: match
data: {"error":"missing form file field 'sample'"}
```

# Example flow

## 1. Index a few songs

```bash
for file in *.wav *.mp3; do
  base="${file%.*}"

  base="${base//_/ }"

  if [[ "$base" == *" - "* ]]; then
    artist="${base%% - *}"
    title="${base#* - }"
  else
    artist=""
    title="$base"
  fi
  echo "Adding $file ..."
  ./trackseek index --title="$title" --artist="$artist" "$file"
done
```

## 2. Test with a sample

```bash
./trackseek match --min-score=400 --threshold=1100 ./match-test.mp3
./trackseek match --min-score=400 --threshold=1100 ./match-test-fail.mp3
```

## 3. Read the result

Possible output:

```text
best match: track_id=17 title="Time doesnt exist" artist="Nortsch" path=./nortsch-time.mp3 score=1261 offset_ms=69474
```

Or with early stop:

```text
best match: track_id=17 title="Time doesnt exist" artist="Nortsch" path=./nortsch-time.mp3 [early stopped] offset_ms=69474
```

## 4. Test the HTTP endpoint

Start the server:

```bash
./trackseek serve
```

Then call the match endpoint:

```bash
curl -N \
  -F "sample=@./match-test.mp3" \
  -F "minScore=400" \
  -F "threshold=1100" \
  http://localhost:8080/match
```

There is also an IDE-friendly request file:

```text
api-test/match.http
```

# Notes

- `.wav` and `.mp3` are supported
- this is a prototype, not a final production matcher
- schema mismatches now fail explicitly instead of recreating existing tables automatically
- the HTTP match endpoint returns SSE with JSON payload data

# Ideas

- better performance for large databases
- make scalable
