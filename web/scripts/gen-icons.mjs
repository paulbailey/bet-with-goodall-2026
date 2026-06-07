// Rasterise the app icon (icons/trophy.svg) into the PNG sizes the web app
// manifest and iOS reference. Run with:
//
//   npm i -D sharp && node scripts/gen-icons.mjs && npm uninstall sharp
//
// sharp is only needed to (re)generate icons, not to build or run the site, so
// it isn't a committed dependency — the rasterised PNGs in public/ are what
// ship. Re-run this whenever icons/trophy.svg changes.
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
import sharp from 'sharp'

const here = dirname(fileURLToPath(import.meta.url))
const svg = readFileSync(join(here, '..', 'icons', 'trophy.svg'))
const out = join(here, '..', 'public')

// "any" icons and the maskable/apple icons all render from the same full-bleed
// source — the trophy sits inside the maskable safe zone, so one composition
// covers every purpose.
const targets = [
  { name: 'pwa-192x192.png', size: 192 },
  { name: 'pwa-512x512.png', size: 512 },
  { name: 'pwa-maskable-512x512.png', size: 512 },
  { name: 'apple-touch-icon.png', size: 180 },
]

for (const { name, size } of targets) {
  await sharp(svg, { density: 384 })
    .resize(size, size)
    .png()
    .toFile(join(out, name))
  console.log(`wrote public/${name} (${size}x${size})`)
}
