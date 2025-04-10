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

import { exec } from "node:child_process";
import { promisify } from "node:util";

import startNavigator from "../basic_navigator_example/.scripts/start.mjs";
import buildWrapperAppTemplate from "../wrapper_app_project/.scripts/build.mjs";
import minimist from "minimist";
import chalk from "chalk";

(async function () {
  const args = minimist(process.argv.slice(2));

  if (!args.platform) {
    throw new Error(`Parameter \`--platform\` not provided.`);
  }

  if (!args.entryUrl) {
    throw new Error(`Parameter \`--entryUrl\` not provided.`);
  }

  if (args.output) {
    throw new Error(`Parameter \`output\` is not supported in the 'start' script`);
  }

  let proxyLocations = {};
  if (args.navigatorToken && args.navigatorPath) {
    proxyLocations = await startNavigator(args);
  } else {
    console.warn(
      chalk.yellow(
        `Parameters \`--navigatorToken\` && \`--navigatorPath\` not provided: skipping navigation example. Building the app directly for "${args.entryDomain}".`,
      ),
    );
  }

  await buildWrapperAppTemplate({ ...args, ...proxyLocations });

  await promisify(exec)(`npm run open:${args.platform}`);
})();
