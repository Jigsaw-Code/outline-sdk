// WIP

import { exec } from "node:child_process";
import { promisify } from "node:util";

import buildWrapperAppTemplate from "./build_wrapper_app_template.mjs";
import minimist from "minimist";

(async function(){
  const args = minimist(process.argv.slice(2));

  await npmRun(`build:sdk_mobileproxy ${args.platform}`);

  await buildWrapperAppTemplate(args);
})();

const npmRun = script => promisify(exec)(`npm run ${script}`);