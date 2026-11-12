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

The mascot and background were generated with the built-in `image_gen` tool. The current mascot uses the approved younger piglet design: a larger head, round cheeks, a smaller snout, a gentle smile, and a tiny brown suit with an orange tie. The existing mascot was the reference image. The approved output was resampled with macOS `sips` to 640 pixels for the app and 64 pixels for the matching favicon. The background is a WebP conversion at quality 88.

| Asset                                       | Dimensions | Format   | Size          |
| ------------------------------------------- | ---------- | -------- | ------------- |
| `apps/web/public/assets/banker-piggy.png`   | 640 × 640  | RGBA PNG | 352,164 bytes |
| `apps/web/public/assets/warm-interior.webp` | 1672 × 941 | RGB WebP | 205,706 bytes |
| `apps/web/public/assets/favicon.png`        | 64 × 64    | RGBA PNG | 6,635 bytes   |

The mascot and favicon preserve actual alpha transparency. The favicon URL carries the design version so browsers can fetch the new icon. The welcome illustration and sidebar use the same mascot file.

## Mascot prompt

```text
Use case: precise-object-edit.
Asset type: Bank Piggy mascot, a transparent square app icon and welcome illustration.
Edit target: Image 1 is the current Bank Piggy mascot. Redesign this same banker pig to look much younger and cuter, like a cheerful little piglet playing dress-up as a banker.
Change: Give him a larger round head relative to a much smaller body, soft plump cheeks, a short little button snout, large gentle dark-brown eyes with warm catchlights, small soft floppy ears, and a sweet small smile. Reduce the mature human-like eyebrows and broad adult shoulders. The expression should feel innocent, friendly, and charming, without the existing large adult-looking open-mouth grin.
Keep: Pink pig identity, chocolate-brown suit jacket, cream shirt, burnt-orange necktie, centered front-facing head and upper torso. Use a tiny softly rounded jacket and a slightly oversized orange tie to make the outfit playful. No visible hands. Preserve the warm brown, cream, orange, and pink brand palette.
Style: Polished dimensional cartoon illustration, soft velvety shading, rounded toy-like forms, a clear simple silhouette that stays readable at 64 pixels. Refined and warm rather than overly shiny plastic.
Composition: One mascot only, comfortably centered in a square with a modest clear margin. The head occupies about three quarters of the character's height.
Background: Genuine transparent alpha, clean antialiased edges, no speckles, no backdrop, no ground plane or cast shadow.
Avoid: Text, logos, coins, props, hats, glasses, accessories, border, watermark, realistic animal fur, human child anatomy, or a full scene.
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
