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

/**
 * @param {string} filepath
 * @returns {Promise<{}>}
 */
export async function getYAMLFileConfig(filepath) {
  try {
    const data = await fs.readFile(filepath, 'utf8')
    
    if (data) {
      const parsedData = YAML.parse(data);

      if (parsedData && typeof parsedData === 'object' && !Array.isArray(parsedData)) {
        // This type assertion may not be 100% guaranteed but for the purposes
        // of this use case should be correct
        return /** @type {{}} */ (parsedData);
      } else {
        console.warn(`${filepath} contained invalid config data:`, parsedData)
      }
    } else {
      console.warn(`${filepath} contained no data`)
    }
  } catch (e) {
    console.warn(`Error loading ${filepath}:`, e)
  }

  return {};
}

/**
 * @param {NodeJS.Process["argv"]} args
 */
export function getCliConfig(args) {
  const dict = minimist(args)
  return {
    ...dict,
    additionalDomains: dict.additionalDomains?.split(',') ?? []
  }
}