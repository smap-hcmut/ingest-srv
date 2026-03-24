# UAP Validation Report - 2026-03-24

## Scope

Validation was run before changing UAP with these live/raw artifacts:

- Script: `scapper-srv/scripts/validate_uap_sources.py`
- Summary: `scapper-srv/output/validation_uap/uap_validation_summary_20260323_225206.json`
- YouTube augmented full flow: `scapper-srv/output/validation_uap/youtube_full_flow_with_transcript_validation_20260323_225120.json`
- Facebook video/photo caption check: `scapper-srv/output/validation_uap/facebook_video_caption_validation_20260323_225116.json`
- TikTok live full flow threshold 0.3: `scapper-srv/output/validation_uap/tiktok_full_flow_threshold03_validation_20260323_225206.json`
- TikTok direct reply probe on historical comments: `scapper-srv/output/validation_uap/tiktok_comment_replies_probe_20260323_225235.json`

## Direct Answers

- YouTube `full_flow` can be extended to include `transcript` without breaking the existing shape.
  - Tested shape: `video`, `detail`, `comments`, `transcript`
  - Both tested videos returned usable transcript text.
- Facebook video post does not show `accessibility_caption` like the tested photo post.
  - Tested video attachment: `type=Video`, `accessibility_caption=null`
  - Tested photo attachment: `type=Photo`, `accessibility_caption` present and looks like alt/OCR text, not transcript.
- Reply body extraction is not reliable in current raw.
  - TikTok live `full_flow` at `threshold=0.3`: no comments returned in 3 tested keywords, therefore no reply bodies.
  - TikTok direct `comment_replies` probe on 5 historical comments with `reply_count > 0`: all returned `replies=[]`.
  - YouTube current raw: `reply_count` exists, reply bodies are absent.
  - Facebook current raw: `replies=null` in tested comments, no reply bodies observed.

## Hot Keyword Coverage Rerun

Additional coverage run used a heuristic mix of tech, sports, entertainment, and auto keywords:

- Summary: `scapper-srv/output/validation_uap/uap_hot_keyword_coverage_summary_20260323_234538.json`
- YouTube raw bundle: `scapper-srv/output/validation_uap/youtube_hot_keyword_full_flow_coverage_raw_20260323_234459.json`
- Facebook raw bundle: `scapper-srv/output/validation_uap/facebook_hot_keyword_full_flow_coverage_raw_20260323_234419.json`
- TikTok raw bundle: `scapper-srv/output/validation_uap/tiktok_hot_keyword_full_flow_coverage_raw_20260323_234538.json`

Coverage highlights:

- YouTube, 10 videos across 5 keywords
  - stable in `10/10`: `detail.title`, `detail.description`, `detail.keywords`, `detail.author_url`, `detail.width`, `detail.height`, `detail.date_published`, `detail.upload_date`
  - transcript usable in `8/10`
  - comments present in `9/10`
  - reply support remains count-only

- Facebook, 15 posts across 5 keywords
  - stable in `15/15`: `post.message`, `post.url`, `post.author.url`, `post.author.avatar_url`, `post.created_time`
  - comments present in `14/15`
  - `comments.author.profile_url` present in `14/15`
  - attachments present in `9/15`
  - `accessibility_caption` present in `4/15`, all from photo attachments
  - tested attachment types: `Photo=4`, `Video=5`
  - video posts with `accessibility_caption`: `0`

- TikTok, 15 posts across 5 keywords
  - stable in `15/15`: `post.description`, `post.hashtags`, `detail.bookmarks_count`, `detail.music_title`, `detail.music_url`, `detail.downloads.video`, `detail.video_resources`
  - subtitle-related fields present in `8/15`: `detail.subtitle_url`, `detail.downloads.subtitle`
  - comments present in `0/15` in this rerun
  - reply support still unavailable in current live raw

## Docs vs Implement

- UAP spec says `platform: tiktok, facebook, youtube`, but current ingest parser only accepts TikTok `full_flow`.
- Ingest dispatch currently uses TikTok `threshold=0.3`.
- TikTok handler default is still `0.5`, so runtime behavior depends on caller.
- Current UAP spec uses `content.subtitle_url`, but raw capability now supports a better normalized text field for TikTok/YouTube:
  - TikTok: downloadable subtitle source via `subtitle_url`
  - YouTube: transcript text via `full_text` and `segments`

## Per Platform

### TikTok

- Live `full_flow` test with `threshold=0.3` on `vinfast vf8`, `iphone 16`, `bia tiger` returned `total_comments=0` for all 3 runs.
- This means current live `full_flow` evidence does not support `COMMENT` or `REPLY` generation from those runs.
- Historical raw sample still shows many comments and non-zero `reply_count`, but current direct `comment_replies` probe returned no bodies for 5 sampled comments.
- TikTok subtitle capability is real.
  - `subtitle_url` can be downloaded as `WEBVTT`
  - `downloads.subtitle` currently returned `401` in earlier direct test

### YouTube

- Augmented `full_flow` worked as additive shape only.
  - Existing keys stayed intact: `video`, `detail`, `comments`
  - New key added cleanly: `transcript`
- Tested videos:
  - `5gnmCqjnQCU`: transcript length `7592`, `56` segments
  - `DPENsCY6WyE`: transcript length `7596`, `50` segments
- Comments currently expose `reply_count` but not reply bodies.
- Raw already has several cross-platform useful fields not in current UAP:
  - `detail.keywords`
  - `detail.author_url`
  - `detail.width`, `detail.height`
  - `date_published`, `upload_date`

### Facebook

- Tested keyword `Theanh28 Entertainment` returned both photo and video posts.
- Tested video attachment did not contain `accessibility_caption`.
- Tested photo attachment did contain `accessibility_caption`, but preview semantics look like generated alt/OCR text, not subtitle/transcript text.
- Current comments raw provides:
  - `author.profile_url`
  - `reaction_count`
  - `reply_count`
  - `replies`
  - `created_time`
- In tested sample, `replies` was effectively unavailable for reply-body flattening.
- Post detail raw also exposes `updated_time`, which current UAP does not use.

## Raw Fields UAP Is Not Using Yet

### Usable Now

- `content.subtitle` from TikTok subtitle text and YouTube transcript text
- `content.title` from TikTok summary title and YouTube title
- `content.keywords` from TikTok summary keywords and YouTube keywords
- `author.profile_url` from Facebook/YouTube
- `temporal.updated_at` from Facebook `updated_time`
- `media.width`, `media.height` from YouTube and some Facebook attachments
- `content.links` via regex on TikTok description, YouTube description, Facebook message/comment

### Needs More Validation

- Facebook `accessibility_caption` as transcript-like content
- Cross-platform `discussion_depth`
- Cross-platform `is_verified`

### Platform-Specific Only

- TikTok `music_title`
- TikTok `music_url`
- TikTok `is_shop_video`
- TikTok `video_resources`
- TikTok comment `sort_extra_score`

## Conclusion

- The cleanest cross-platform transcript path is:
  - TikTok subtitle text from downloaded `WEBVTT`
  - YouTube transcript text from `full_text` or merged `segments`
- Facebook should not be mapped to transcript/subtitle yet.
- Current reply-body support is not strong enough to promise `uap_type=REPLY` for any platform right now.
- UAP spec should be updated to reflect actual parser support, and core fields should be renamed toward cross-platform semantics before adding Facebook/YouTube parsing.
