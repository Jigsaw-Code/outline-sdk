import { promises as fs } from 'node:fs'
import path from 'node:path'

import minimist from 'minimist'
import YAML from 'yaml'


export const DEFAULT_CONFIG = {
  output: path.join(process.cwd(), 'output'),
  smartDialerConfig: JSON.stringify({
    dns: [
      {
        https: { name: '9.9.9.9' }
      }
    ],
    tls: [
      '',
      'split:1',
      'split:2',
      'tlsfrag:1'
    ],   
  })
}

export async function getYAMLFileConfig(filepath) {
  try {
    const data = await fs.readFile(filepath, 'utf8')
    
    if (data) {
      return YAML.parse(data)
    }
  } catch (e) {
    if ('ENOENT' == e.code) {
      return {}
    }
  }
}

export function getCliConfig(args) {
  const dict = minimist(args)
  return {
    ...dict,
    additionalDomains: dict.additionalDomains?.split(',') ?? []
  }
}