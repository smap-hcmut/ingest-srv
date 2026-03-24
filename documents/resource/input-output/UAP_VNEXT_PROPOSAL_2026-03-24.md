# UAP vNext Proposal - 2026-03-24

## Design Goal

Core UAP should prefer fields that map cleanly across TikTok, Facebook, and YouTube. Platform-specific signals should move out of the core object into `platform_meta` or stay only in raw storage.

## Proposed Core Changes

### Edit

- `content.subtitle_url` -> `content.subtitle`
  - Type: `string`
  - Meaning: normalized subtitle/transcript text, with timestamps and cue markers removed.
  - Why: works for TikTok subtitle and YouTube transcript using the same semantics.
  - Mapping:
    - TikTok: download `subtitle_url`, parse `WEBVTT`, merge text lines
    - YouTube: `full_text`, fallback merge `segments[].text`
    - Facebook: leave empty until validated

- `content.summary_title` -> `content.title`
  - Type: `string`
  - Meaning: best available content title.
  - Mapping:
    - TikTok: `detail.summary.title`
    - YouTube: `detail.title` or `video.title`
    - Facebook: optional, often empty

- `content.tiktok_keywords` -> `content.keywords`
  - Type: `string[]`
  - Meaning: platform-provided or AI-extracted semantic keywords.
  - Mapping:
    - TikTok: `detail.summary.keywords`
    - YouTube: `detail.keywords`
    - Facebook: empty for now

- `engagement.bookmarks` -> `engagement.saves`
  - Type: `int|null`
  - Meaning: user save/bookmark behavior in a platform-neutral name.

### Add

- `author.profile_url`
  - Type: `string`
  - Why: already available in Facebook/YouTube raw, useful for drill-through and identity QA.

- `content.links`
  - Type: `string[]`
  - Why: regex-extractable from post/comment text on all three platforms.

- `temporal.updated_at`
  - Type: `string`
  - Why: Facebook raw already exposes `updated_time`.

- `media.width`
  - Type: `int|null`
  - Why: already present in YouTube detail and some Facebook attachments.

- `media.height`
  - Type: `int|null`
  - Why: already present in YouTube detail and some Facebook attachments.

- `platform_meta`
  - Type: `object`
  - Why: keeps platform-specific signals without polluting the cross-platform core schema.

### Move Out Of Core

- `content.music_title`
  - Move to `platform_meta.tiktok.music_title`
  - Reason: TikTok-specific trend signal, no clear Facebook/YouTube equivalent.

- `content.music_url`
  - Move to `platform_meta.tiktok.music_url`
  - Reason: TikTok-specific media asset.

- `content.is_shop_video`
  - Move to `platform_meta.tiktok.is_shop_video`
  - Reason: commerce semantics are not aligned across the 3 tested raw shapes.

- `engagement.sort_score`
  - Move to `platform_meta.tiktok.sort_score`
  - Reason: this score is specific to TikTok comment ranking logic.

## Proposed Reply Policy

- Only emit `uap_type=REPLY` when raw includes actual reply bodies.
- If raw only exposes `reply_count`, keep it in `engagement.reply_count`.
- Do not fabricate reply records from counts.

## Suggested Content Shape

```json
{
  "content": {
    "text": "raw primary text",
    "title": "best available title",
    "subtitle": "normalized transcript text",
    "hashtags": ["..."],
    "keywords": ["..."],
    "links": ["https://..."]
  }
}
```

## Suggested Media Shape

```json
{
  "media": [
    {
      "type": "video",
      "url": "playable or canonical media URL",
      "download_url": "optional direct download URL",
      "thumbnail": "thumbnail URL",
      "duration": 84,
      "width": 1920,
      "height": 1080
    }
  ]
}
```

## Mapping Notes By Platform

### TikTok

- Keep support for `POST`, `COMMENT`, `REPLY`, but only when raw actually has data.
- Use subtitle text instead of subtitle URL in core UAP.
- Keep `music_*`, `is_shop_video`, and `video_resources` in `platform_meta`.

### YouTube

- Extend `full_flow` with `transcript` as additive field.
- Map transcript into `content.subtitle`.
- Map `author_url` into `author.profile_url`.
- Map `keywords`, `width`, `height`, `date_published`.

### Facebook

- Add parser only for fields proven stable in raw:
  - `message`
  - `author`
  - `reaction_count`
  - `comment_count`
  - `share_count`
  - `created_time`
  - `updated_time`
  - `attachments`
- Do not map `accessibility_caption` into `content.subtitle` yet.
- If later a video-specific transcript/caption field is confirmed, then map it into `content.subtitle`.

## Implementation Order

1. Update UAP spec names toward cross-platform fields.
2. Add subtitle normalization utility for TikTok `WEBVTT` and YouTube transcripts.
3. Extend YouTube `full_flow` with additive `transcript`.
4. Add Facebook and YouTube parsers in ingest.
5. Keep TikTok-specific fields in `platform_meta` instead of core UAP.

## Non-Goals For This Round

- Do not add Facebook transcript mapping until validated on real video raw.
- Do not guarantee `REPLY` support until reply bodies are consistently available.
- Do not remove raw storage of original URLs and platform-specific objects.
