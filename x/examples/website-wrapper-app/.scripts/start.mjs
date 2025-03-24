// WIP

import { exec } from "node:child_process";
import { promisify } from "node:util";

import startNavigationProxy from "./start_navigation_proxy.mjs";
import buildWrapperAppTemplate from "./build_wrapper_app_template.mjs";
import minimist from "minimist";
import chalk from "chalk";

(async function () {
  const args = minimist(process.argv.slice(2));

  if (!args.platform) {
    throw new Error(`Parameter \`--platform\` not provided.`);
  }

  if (!args.entryDomain) {
    throw new Error(`Parameter \`--entryDomain\` not provided.`);
  }

  let proxyLocations = {};
  if (args.proxyToken && args.proxyNavigationPath) {
    proxyLocations = await startNavigationProxy(args);
  } else {
    console.warn(chalk.yellow(`Parameters \`--proxyToxen\` && \`--proxyNavigationPath\` not provided: skipping navigation proxy. Building the app directly for "${args.entryDomain}".`));
  }

  await buildWrapperAppTemplate({ ...args, ...proxyLocations });

  // TODO: not working
  await promisify(exec)(`npm run open:${args.platform}`);
})();
