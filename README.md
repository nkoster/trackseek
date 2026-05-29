# trackseek

trackseek is a small Shazam-like lab project in Go.

The idea is simple:

- index audio files into a SQLite database
- store audio fingerprints
- take a short sample
- try to find the right track back

Right now this is a local CLI tool.
Later this can grow into a REST service.

# What it does

- reads audio from `.wav` and `.mp3`
- extracts peaks from the spectrum
- builds fingerprint hashes
- stores fingerprints in SQLite
- matches a sample against stored tracks

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
./trackseek match --min-score=80 ./sample.mp3
```

With early stop for clear matches:

```bash
./trackseek match --min-score=80 --threshold=280 ./sample.mp3
```

# Matching flags

## `--min-score`

This is the minimum score needed to accept a match.

Example:

- if best score is `57`
- and `--min-score=80`
- then result is `no matching track found`

## `--threshold`

This is an early stop value.

If a candidate reaches this score during matching,
the matcher stops early and returns that result.

This is useful when you want to save time and resources.

If this happens, the output shows:

```text
[early stopped]
```

That means the match was accepted early.
It does **not** mean the shown score is the full final score.

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
./trackseek match --min-score=80 --threshold=280 ./match-test.mp3
./trackseek match --min-score=80 --threshold=280 ./match-test-fail.mp3
```

## 3. Read the result

Possible output:

```text
best match: track_id=17 title="Time doesnt exist" artist="Nortsch" path=./nortsch-time.mp3 score=289 offset_ms=69474
```

Or with early stop:

```text
best match: track_id=17 title="Time doesnt exist" artist="Nortsch" path=./nortsch-time.mp3 [early stopped] offset_ms=69474
```

# Notes

- `.wav` and `.mp3` are supported
- this is a prototype, not a final production matcher
- the schema code may recreate old tables when the schema changes

# Ideas

- REST API
- upload endpoint
- better performance for large databases
- make scalable
