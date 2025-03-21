// WIP

import { exec } from "node:child_process";
import { promisify } from "node:util";

import startWebsiteProxy from "./start_website_proxy.mjs";
import buildWrapperAppTemplate from "./build_wrapper_app_template.mjs";
import minimist from "minimist";

(async function () {
  const args = minimist(process.argv.slice(2));

  await npmRun(`build:sdk_mobileproxy ${args.platform}`);

  const proxyLocations = await startWebsiteProxy(args);

  await buildWrapperAppTemplate({ ...args, ...proxyLocations });

  await npmRun(`open:${args.platform}`);
})();

const npmRun = (script) => promisify(exec)(`npm run ${script}`);
