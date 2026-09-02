# OpenLicensd Brand Assets

Design source files for the OpenLicensd visual identity.

## Logo

| File | Use |
|------|-----|
| `logo-light.svg` | Combined mark + wordmark on light backgrounds (README, docs) |
| `logo-dark.svg` | Combined mark + wordmark on dark backgrounds (white + accent `#2F6FFF`) |
| `mark-light.svg` | Mark on light backgrounds (navy `#111C34` + accent `#2F6FFF`) |
| `mark-dark.svg` | Mark on dark backgrounds (white + accent `#2F6FFF`) |
| `wordmark-light.svg` | Full wordmark on light backgrounds |
| `wordmark-dark.svg` | Full wordmark on dark backgrounds |

The admin UI renders these as Vue components (`ui/components/BrandMark.vue`, `ui/components/BrandWordmark.vue`) using `currentColor` for theme-dependent paths and `#2F6FFF` for the accent.

The Vue components use cropped viewBoxes for optical centering in lockups (mark + wordmark side by side). `BrandWordmark.vue` crops to glyph ink and top-pads by the descender depth so `items-center` aligns cap height with the mark. A straight re-export from Figma will reintroduce empty vertical padding and regress lockup sizing/centering.

## Typography

**Space Grotesk** (SIL Open Font License 1.1)

- Titles: Medium (500), letter-spacing −2% (`tracking-brand` / `-0.02em`)
- Self-hosted in `ui/public/fonts/` (Regular, Medium, SemiBold, Bold)
- Source TTF files are kept in `docs/brand/fonts/` for reference

## Color Palette

| Purpose | Hex |
|---------|-----|
| Primary Blue | `#2F6FFF` |
| Accent Blue | `#4D8DFF` |
| Dark Navy | `#111C34` |
| Midnight | `#1C2438` |
| Background | `#F7F9FC` |
| Surface | `#FFFFFF` |
| Border | `#E6EBF3` |
| Secondary Text | `#6E7C93` |
| Primary Text | `#1A2238` |

In the UI, these map to Tailwind `brand-*` and `navy-*` scales plus Nuxt UI semantic tokens (`text-highlighted`, `text-muted`, `bg-default`, `border-default`, etc.) defined in `ui/assets/css/main.css`.

## Favicon

`ui/public/favicon.svg` — full-color mark on a square viewBox.

## License

Space Grotesk is licensed under the [SIL Open Font License 1.1](https://scripts.sil.org/OFL). See `ui/public/fonts/OFL.txt`.
