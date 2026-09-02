import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.dirname(fileURLToPath(import.meta.url))
const publicDir = path.join(root, '..', 'public')

fs.mkdirSync(publicDir, { recursive: true })

for (const item of ['brand', 'screenshots', 'openapi.yaml']) {
  const source = path.join(root, '..', item)
  const target = path.join(publicDir, item)

  fs.rmSync(target, { recursive: true, force: true })
  fs.cpSync(source, target, { recursive: true })
}
