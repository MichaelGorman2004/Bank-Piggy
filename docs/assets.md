# Generated assets

## Payment provider logos

The 100 service options use local SVG files in `apps/web/public/assets/providers`.
Generic bill, subscription, and car icons cover other payments.
`scripts/generate-provider-icons.mjs` builds the selected marks from Simple Icons
16.37.0 and four checked-in source files from its 11.15.0 release. Some service
options use their parent company's mark.

The generator rejects active content and external references. Catalog lookup
resolves each provider before the app builds its local image path. Run
`npm run providers:generate` to rebuild the files or `npm run providers:check` to
verify the committed output. The main package declares CC0-1.0. The archived
sources are [Amazon](https://github.com/simple-icons/simple-icons/blob/11.15.0/icons/amazon.svg),
[Microsoft](https://github.com/simple-icons/simple-icons/blob/11.15.0/icons/microsoft.svg),
[Adobe](https://github.com/simple-icons/simple-icons/blob/11.15.0/icons/adobe.svg), and
[Slack](https://github.com/simple-icons/simple-icons/blob/11.15.0/icons/slack.svg).

## Banker piggy and background

Both original images were generated with the built-in `image_gen` tool. No API key or CLI fallback was used. The selected outputs are saved in this repository. The final mascot is a 640-pixel Lanczos downscale of the first built-in cleanup edit. The favicon is a 64-pixel Lanczos downscale of that final mascot. The background is a WebP conversion at quality 88.

| Asset                                       | Dimensions | Format   | Size          |
| ------------------------------------------- | ---------- | -------- | ------------- |
| `apps/web/public/assets/banker-piggy.png`   | 640 × 640  | RGBA PNG | 406,153 bytes |
| `apps/web/public/assets/warm-interior.webp` | 1672 × 941 | RGB WebP | 205,706 bytes |
| `apps/web/public/assets/favicon.png`        | 64 × 64    | RGBA PNG | 8,648 bytes   |

The mascot and favicon contain actual alpha transparency. Both generated images and the favicon were visually checked. The mascot stays readable at 64 pixels. Two built-in cleanup edits preserved the pig. The first edit was selected; the second increased stray pixels and was rejected. Small disconnected pixels remain at alpha 1/255. The image viewer exaggerates their red color; normal compositing makes them nearly invisible. No manual alpha removal was applied.

## Mascot prompt

```text
Use case: logo-brand
Asset type: Bank Piggy finance dashboard mascot, square icon and larger welcome illustration.
Primary request: A friendly pink cartoon banker pig, head and upper torso, with a brown suit jacket, cream shirt, and burnt orange tie. Charming rounded polished illustration, expressive kind eyes, small ears, simple readable face, soft dimensional shading and crisp silhouette. Centered front-facing portrait, comfortably fits a square with small clear margins, readable at 48 to 80 pixels.
Scene/backdrop: Genuine transparent alpha background; no backdrop, no ground plane.
Color palette: Warm pink skin, chocolate brown jacket, cream shirt, burnt orange tie.
Constraints: Only the pig and its clothing. No text, numbers, logos, coins, glasses, hats, props, frame, border, or watermark. Actual alpha transparency around the complete silhouette.
```

Tool option: `transparent_background: true`.

## Background prompt

```text
Use case: photorealistic-natural
Asset type: Wide background photograph for a calm personal finance dashboard.
Primary request: A realistic warm midcentury home interior, walnut furniture, warm cream plaster walls, soft orange sunset indirect light, a potted plant, and a large window. Architectural interior photography with a wide horizontal composition, calm tasteful lived-in atmosphere, subtle real material texture, natural light. Include a low walnut sideboard and cream seating, with generous quiet wall and floor areas. Gentle depth of field and subdued detail so foreground dashboard cards remain the focus.
Lighting/mood: Warm calm late afternoon sunset, soft contrast, muted natural browns and cream, orange glow.
Constraints: No people, text, UI, logos, watermark, office equipment, visible screens, or dramatic high-contrast patterns. Wide landscape image.
```

Tool option: `transparent_background: false`.

## Selected cleanup edit prompt

```text
Use case: background-extraction
Edit target: The provided Bank Piggy mascot PNG.
Primary request: Clean only the transparent background and silhouette edge. Remove every disconnected red, white, or colored speckle, stray pixel cluster, and halo outside the pig's main silhouette, especially the speckles above the ears. Output a clean polished cutout with smooth antialiased edges and actual alpha transparency.
Invariants: Preserve this exact pig identity and face, pink color, small ears, joyful expression, brown suit jacket, cream shirt, burnt orange tie, front-facing head/upper-torso composition and current proportions. Do not redesign, add or remove features, or change clothing, shading, pose or expression. Keep the complete pig centered in a square with a small transparent margin around it. No text, logos, watermark, props, added background or shadow plane.
```

Tool option: `transparent_background: true`. The original local PNG was the edit target.
