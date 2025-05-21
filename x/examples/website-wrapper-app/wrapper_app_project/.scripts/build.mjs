// Copyright 2025 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { exec, execFile } from 'node:child_process'
import { promises as fs } from 'node:fs'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { promisify } from 'node:util'

import chalk from 'chalk'
import { glob } from 'glob'
import handlebars from 'handlebars'

import { getCliConfig, getYAMLFileConfig, DEFAULT_CONFIG } from './config.mjs'
import { resolveConfiguration, zip } from './util.mjs'

const TEMPLATE_DIR = path.join(process.cwd(), 'wrapper_app_project/template');


/* See https://stackoverflow.com/questions/57838022/detect-whether-es-module-is-run-from-command-line-in-node*/
if (import.meta.url !== pathToFileURL(`${process.argv[1]}`).href) {
  throw new Error('Build script must be run from the cli')
}

const config = resolveConfiguration({
  ...DEFAULT_CONFIG,
  ...(await getYAMLFileConfig('config.yaml')),
  ...getCliConfig(process.argv)
})

const APP_TARGET_DIR = path.resolve(config.output, config.appName)
const APP_TARGET_ZIP = path.resolve(config.output, `${config.appName}.zip`)

const SDK_TARGET_BIN = path.resolve(config.output, 'mobileproxy')
const SDK_TARGET_DIR = path.resolve(APP_TARGET_DIR, 'mobileproxy')

try {
  await fs.access(SDK_TARGET_BIN, fs.constants.F_OK)
} catch (err) {
  console.log(chalk.bgGreen(`Building the Outline SDK mobileproxy library for ${config.platform}...`))
  await promisify(execFile)('npm', ['run', 'build:mobileproxy', config.platform, config.output])
}

const sourceFilepaths = await glob(
  path.join(TEMPLATE_DIR, '**', '*'),
  {
    nodir: true,
    dot: true,
  },
)

console.log(chalk.bgGreen('Building the wrapper app project from template...'))

for (const sourceFilepath of sourceFilepaths) {
  const destinationFilepath = path.join(APP_TARGET_DIR, path.relative(TEMPLATE_DIR, sourceFilepath))

  // ensure directory
  await fs.mkdir(path.dirname(destinationFilepath), { recursive: true })

  if (sourceFilepath.endsWith('.handlebars')) {
    console.log(chalk.white(`render ${sourceFilepath}`))
    const template = handlebars.compile(await fs.readFile(sourceFilepath, 'utf8'))
    await fs.writeFile(destinationFilepath.replace(/\.handlebars$/, ''), template(config), 'utf8')
  } else {
    console.log(chalk.gray(`copy ${sourceFilepath}`))
    await fs.cp(sourceFilepath, destinationFilepath)
  }
}

console.log(chalk.green('Copying mobileproxy files into the project...'))
await fs.cp(SDK_TARGET_BIN, SDK_TARGET_DIR, { recursive: true })

console.log(chalk.green('Installing external dependencies for the project...'))
await promisify(exec)(`
  cd ${APP_TARGET_DIR.replaceAll(/\s+/g, '\\ ')}
  npm install --no-warnings
  npx cap sync ${config.platform}
`)

console.log(chalk.green(`Zipping project to ${chalk.blue(APP_TARGET_ZIP)}...`))
await zip(APP_TARGET_DIR, APP_TARGET_ZIP)

console.log(chalk.bgGreen('Project ready!'))

if ('android' === config.platform) {
  console.log(chalk.white('To open your project in Android Studio:'))
  console.log(chalk.gray(`  cd ${APP_TARGET_DIR.replaceAll(/\s+/g, '\\ ')}`))
  console.log(chalk.gray('  npm run open:android'))
} else if ('ios' === config.platform) {
  console.log(chalk.white('To open your project in Xcode:)'))
  console.log(chalk.gray(`  cd ${APP_TARGET_DIR.replaceAll(/\s+/g, '\\ ')}`))
  console.log(chalk.gray('  npm run open:ios'))
}