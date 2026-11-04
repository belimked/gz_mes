/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { auditAction } from '@/utils/auditActions'

describe('audit action names', () => {
  it('names every mutating OpenAPI route', () => {
    const yaml = readFileSync('../api/openapi.yaml', 'utf8')
    let path = ''
    for (const line of yaml.split('\n')) {
      const match = /^ {2}(\/[^:]+):$/.exec(line)
      if (match) path = match[1]!
      const method = /^ {4}(post|put|patch|delete):$/.exec(line)?.[1]?.toUpperCase()
      if (method) expect(auditAction(method, path), `${method} ${path}`).not.toBe(`${method} ${path}`)
    }
  })
})
