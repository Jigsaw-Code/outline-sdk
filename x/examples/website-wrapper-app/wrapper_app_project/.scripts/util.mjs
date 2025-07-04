import fs from 'node:fs'

import archiver from 'archiver'

/** @typedef {import("./types.mjs").Config} Config */

/**
 * @param {Record<string, unknown>} config
 * @returns {Config}
 */
export function resolveConfiguration(config) {
  const resolved = {};
  
  if (typeof config.entryUrl !== "string") {
    throw new Error(`"entryUrl" parameter not provided in parameters or YAML configuration.`);
  }
  if (typeof config.platform !== "string") {
    throw new Error(`"platform" parameter not provided in parameters or YAML configuration.`);
  }
  
  resolved.entryDomain = new URL(config.entryUrl).hostname;
  resolved.output = new URL(config.entryUrl).hostname;
  resolved.platform = config.platform;

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
  
  resolved.additionalDomains = (
    Array.isArray(config.additionaldomain) &&
    config.additionaldomain.every((item) => typeof item === "string")
  )
    ? config.additionaldomain
    : []
  resolved.domainList = [resolved.entryDomain, ...resolved.additionalDomains].join('\n')
  resolved.smartDialerConfig = Buffer.from(JSON.stringify(config.smartDialerConfig)).toString('base64')

  return resolved;
}

/**
 * @param {string} root
 * @param {string} destination
 */
export function zip(root, destination) {
  const job = archiver('zip', { zlib: { level: 9 } })
  const destinationStream = fs.createWriteStream(destination)

  return new Promise((resolve, reject) => {
    job.directory(root, false)
    job.pipe(destinationStream)
  
    // @ts-expect-error (this appears to be harmless)
    destinationStream.on('close', resolve)
  
    job.on('error', reject)
    destinationStream.on('error', reject)
  
    job.finalize()
  });
}