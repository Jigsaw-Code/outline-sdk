// WIP

import { exec } from "node:child_process";
import { promisify } from "node:util";

import startWebsiteProxy from "./start_website_proxy.mjs";
import buildWrapperAppTemplate from "./build_wrapper_app_template.mjs";
import minimist from "minimist";

(async function(){
  const args = minimist(process.argv.slice(2));

  await npmRun(`build:sdk_mobileproxy ${args.platform}`);

  if (args.proxyToken) {
    const proxyLocations = await startWebsiteProxy(args);
    
    await buildWrapperAppTemplate({...args, ...proxyLocations});  
  } else {
    await buildWrapperAppTemplate(args);
  }

  await npmRun(`open:${platform}`);
})();

const npmRun = script => promisify(exec)(`npm run ${script}`);