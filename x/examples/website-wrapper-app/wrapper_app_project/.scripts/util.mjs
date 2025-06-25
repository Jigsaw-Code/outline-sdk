import fs from 'node:fs'

import archiver from 'archiver'


export function resolveConfiguration(config) {
  
  if (!config.platform) {
    throw new Error(`Parameter \`--platform\` not provided.`);
  }
  
  if (!config.entryUrl) {
    throw new Error(`Parameter \`--entryUrl\` not provided.`);
  }
  
  const resolved = {
    entryDomain: new URL(config.entryUrl).hostname,
  }

  if (!config.appId) {
    // Infer an app ID from the entry domain by reversing it (e.g. `www.example.com` becomes `com.example.www`)
    // It must be lower case, and hyphens are not allowed.
    resolved.appId = resolved.entryDomain
      .replaceAll('-', '')
      .toLocaleLowerCase()
      .split('.')
      .reverse()
      .join('.')
  }

  if (!config.appName) {
    // Infer an app name from the base entry domain part by title casing the root domain:
    // (e.g. `www.my-example-app.com` becomes "My Example App")
    resolved.appName = resolved.entryDomain
      .split('.')
      .reverse()[1]
      .split(/[-_]+/)
      .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
      .join(' ')
  }
  
  resolved.additionalDomains = config.additionaldomain ?? []
  resolved.domainList = [resolved.entryDomain, ...resolved.additionalDomains].join('\n')
  resolved.smartDialerConfig = Buffer.from(JSON.stringify(config.smartDialerConfig)).toString('base64')

  return {
    ...config,
    ...resolved
  }
}

export function zip(root, destination) {
  const job = archiver('zip', { zlib: { level: 9 } })
  const destinationStream = fs.createWriteStream(destination)

  return new Promise((resolve, reject) => {
    job.directory(root, false)
    job.pipe(destinationStream)
  
    destinationStream.on('close', resolve)
  
    job.on('error', reject)
    destinationStream.on('error', reject)
  
    job.finalize()
  });
}