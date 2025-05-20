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

const TEMPLATE = path.join(process.cwd(), 'wrapper_app_project/template');

import { getCliConfig, getFileConfig, DEFAULT_CONFIG } from './config.mjs'
import { resolveConfiguration, zip } from './util.mjs'

/* See https://stackoverflow.com/questions/57838022/detect-whether-es-module-is-run-from-command-line-in-node*/
if (import.meta.url !== pathToFileURL(`${process.argv[1]}`).href) {
  throw new Error('Build script must be run from the cli')
}

const config = resolveConfiguration({
  ...DEFAULT_CONFIG,
  ...(await getFileConfig('config.yaml')),
  ...getCliConfig(process.argv)
})

const WRAPPER_APP_OUTPUT_TARGET = path.resolve(config.output, config.appName)
const WRAPPER_APP_OUTPUT_ZIP = path.resolve(config.output, `${config.appName}.zip`)

const SDK_MOBILEPROXY_OUTPUT_TARGET = path.resolve(config.output, 'mobileproxy')
const WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR = path.resolve(WRAPPER_APP_OUTPUT_TARGET, 'mobileproxy')

try {
  await fs.access(SDK_MOBILEPROXY_OUTPUT_TARGET, fs.constants.F_OK)
} catch (err) {
  console.log(chalk.green(`Building the Outline SDK mobileproxy library for ${config.platform}...`))
  await promisify(execFile)('npm', ['run', 'build:mobileproxy', config.platform, config.output])
}

const sourceFilepaths = await glob(
  path.join(TEMPLATE, '**', '*'),
  {
    nodir: true,
    dot: true,
  },
)

console.log(chalk.green('Building project from template...'))

for (const sourceFilepath of sourceFilepaths) {
  const destinationFilepath = path.join(WRAPPER_APP_OUTPUT_TARGET, path.relative(TEMPLATE, sourceFilepath))

  // ensure directory
  await fs.mkdir(path.dirname(destinationFilepath), { recursive: true })

  if (!sourceFilepath.endsWith('.handlebars')) {
    console.log(chalk.gray(`copy ${sourceFilepath}`))
    await fs.cp(sourceFilepath, destinationFilepath)
  } else {
    console.log(chalk.blue(`render ${sourceFilepath}`))
    const template = handlebars.compile(await fs.readFile(sourceFilepath, 'utf8'))
    await fs.writeFile(destinationFilepath.replace(/\.handlebars$/, ''), template(config), 'utf8')
  }
}

console.log(chalk.yellow('Copying mobileproxy files into the project...'))
await fs.cp(SDK_MOBILEPROXY_OUTPUT_TARGET, WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR, { recursive: true })

console.log(chalk.yellow('Installing external dependencies for the project...'))
await promisify(exec)(`
  cd ${WRAPPER_APP_OUTPUT_TARGET.replaceAll(' ', '\\ ')}
  npm install --no-warnings
  npx cap sync ${config.platform}
`)

console.log(chalk.gray(`Zipping project to ${chalk.blue(WRAPPER_APP_OUTPUT_ZIP)}...`))
await zip(WRAPPER_APP_OUTPUT_TARGET, WRAPPER_APP_OUTPUT_ZIP)

console.log(chalk.bgGreen('Project ready!'))