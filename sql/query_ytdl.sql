-- encoding: utf-8

-- name: GetYtDlpDbCache :one
SELECT *
FROM yt_dl_results
WHERE url = $1
  AND audio_only = $2
  AND resolution = $3;

-- name: UpdateYtDlpCache :exec
INSERT INTO yt_dl_results
(url, audio_only, resolution, file_id, title, description, uploader, upload_count)
VALUES ($1, $2, $3, $4, $5, $6, $7, 1)
ON CONFLICT (url, audio_only, resolution) DO UPDATE
    SET file_id=EXCLUDED.file_id,
        title=EXCLUDED.title,
        description=EXCLUDED.description,
        uploader=EXCLUDED.uploader;

-- name: IncYtDlUploadCount :exec
UPDATE yt_dl_results
SET upload_count=upload_count + 1
WHERE file_id = $1;
