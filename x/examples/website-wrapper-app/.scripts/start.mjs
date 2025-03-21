// WIP

import { exec } from "node:child_process";
import { promisify } from "node:util";

import startWebsiteProxy from "./start_website_proxy.mjs";
import buildWrapperAppTemplate from "./build_wrapper_app_template.mjs";
import minimist from "minimist";

(async function () {
  const args = minimist(process.argv.slice(2));

  const proxyLocations = await startWebsiteProxy(args);

  await buildWrapperAppTemplate({ ...args, ...proxyLocations });

  // TODO: not working
  await promisify(exec)(`npm run open:${args.platform}`);
})();
