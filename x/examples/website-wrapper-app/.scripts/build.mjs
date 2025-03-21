import buildWrapperAppTemplate from "./build_wrapper_app_template.mjs";
import minimist from "minimist";

(async function(){
  const args = minimist(process.argv.slice(2));

  await buildWrapperAppTemplate(args);
})();
