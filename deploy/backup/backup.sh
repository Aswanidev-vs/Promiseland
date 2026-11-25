#!/bin/sh
# PromisedLand backup: consistent SQLite snapshot + media archive.
#
# Usage: backup.sh <destination-dir>
# Env:   KEEP_DAYS (default 14), S3_BUCKET (optional), S3_PREFIX (optional),
#        DATA_DIR (default /srv/promisedland/data), DB_NAME (promisedland.db),
#        MEDIA_SUBDIR (media)
#
# Requires only a POSIX shell plus sqlite3 and tar. The live database is never
# copied: sqlite3's online backup API produces a single consistent file even
# while the API is writing in WAL mode.
set -eu

if [ "$#" -ne 1 ] || [ -z "$1" ]; then
	echo "usage: $0 <destination-dir>" >&2
	exit 2
fi

DEST=$1
DATA_DIR=${DATA_DIR:-/srv/promisedland/data}
DB_NAME=${DB_NAME:-promisedland.db}
MEDIA_SUBDIR=${MEDIA_SUBDIR:-media}
KEEP_DAYS=${KEEP_DAYS:-14}
S3_BUCKET=${S3_BUCKET:-}
S3_PREFIX=${S3_PREFIX:-promisedland}

DB_PATH=$DATA_DIR/$DB_NAME
MEDIA_DIR=$DATA_DIR/$MEDIA_SUBDIR

command -v sqlite3 >/dev/null 2>&1 || {
	echo "error: sqlite3 CLI not found on PATH" >&2
	exit 1
}
command -v tar >/dev/null 2>&1 || {
	echo "error: tar not found on PATH" >&2
	exit 1
}
[ -f "$DB_PATH" ] || {
	echo "error: database not found: $DB_PATH" >&2
	exit 1
}
# The media directory is created lazily on the first upload, so it is legitimately
# absent on a fresh install; say so rather than failing or inventing an archive.
HAVE_MEDIA=1
if [ ! -d "$MEDIA_DIR" ]; then
	HAVE_MEDIA=0
	echo "warning: media directory not found: $MEDIA_DIR (nothing uploaded yet?)"
fi

mkdir -p "$DEST" || {
	echo "error: cannot create destination: $DEST" >&2
	exit 1
}
DEST=$(cd "$DEST" && pwd) || {
	echo "error: cannot enter destination: $DEST" >&2
	exit 1
}

# Refuse to write a backup next to the data it backs up.
case $DEST in
	"$DATA_DIR" | "$DATA_DIR"/*)
		echo "error: destination $DEST is inside the data directory $DATA_DIR" >&2
		exit 1
		;;
esac

case $KEEP_DAYS in
	'' | *[!0-9]*)
		echo "error: KEEP_DAYS must be a non-negative integer, got: $KEEP_DAYS" >&2
		exit 1
		;;
esac

STAMP=$(date -u +%Y%m%dT%H%M%SZ)
DB_OUT=$DEST/promisedland-$STAMP.db
MEDIA_OUT=$DEST/promisedland-media-$STAMP.tar.gz

echo "source database : $DB_PATH"
echo "source media    : $MEDIA_DIR"
echo "destination     : $DEST"
echo "retention       : ${KEEP_DAYS} day(s)"

TMP_DB=$DEST/.promisedland-$STAMP.db.tmp
trap 'rm -f "$TMP_DB"' EXIT HUP INT TERM

echo "snapshotting database via sqlite3 .backup ..."
sqlite3 "$DB_PATH" ".backup '$TMP_DB'"
[ -s "$TMP_DB" ] || {
	echo "error: sqlite3 produced an empty snapshot" >&2
	exit 1
}
sqlite3 "$TMP_DB" 'PRAGMA integrity_check;' | grep -q '^ok$' || {
	echo "error: snapshot failed integrity_check" >&2
	exit 1
}
mv "$TMP_DB" "$DB_OUT"
trap - EXIT HUP INT TERM
gzip -f "$DB_OUT"
echo "wrote $DB_OUT.gz"

echo "archiving media ..."
if [ "$HAVE_MEDIA" -eq 1 ]; then
	tar -czf "$MEDIA_OUT" -C "$DATA_DIR" "$MEDIA_SUBDIR"
	echo "wrote $MEDIA_OUT"
else
	echo "media archive: SKIPPED (no media directory at $MEDIA_DIR)"
fi

if [ -n "$S3_BUCKET" ]; then
	command -v aws >/dev/null 2>&1 || {
		echo "error: S3_BUCKET is set but the aws CLI is not installed" >&2
		exit 1
	}
	echo "syncing to s3://$S3_BUCKET/$S3_PREFIX ..."
	aws s3 cp "$DB_OUT.gz" "s3://$S3_BUCKET/$S3_PREFIX/$(basename "$DB_OUT.gz")" >/dev/null
	echo "wrote s3://$S3_BUCKET/$S3_PREFIX/$(basename "$DB_OUT.gz")"
	if [ "$HAVE_MEDIA" -eq 1 ]; then
		aws s3 cp "$MEDIA_OUT" "s3://$S3_BUCKET/$S3_PREFIX/$(basename "$MEDIA_OUT")" >/dev/null
		echo "wrote s3://$S3_BUCKET/$S3_PREFIX/$(basename "$MEDIA_OUT")"
	fi
else
	echo "S3 offsite copy: NOT CONFIGURED (set S3_BUCKET to enable). Backups are local only."
fi

# Retention deletes only our own backup files in the destination directory.
if [ "$KEEP_DAYS" -gt 0 ]; then
	find "$DEST" -maxdepth 1 -type f \
		\( -name 'promisedland-*.db.gz' -o -name 'promisedland-media-*.tar.gz' \) \
		-mtime +"$KEEP_DAYS" -print -delete
	echo "retention: removed backups older than ${KEEP_DAYS} day(s) from $DEST"
fi

echo "backup complete"