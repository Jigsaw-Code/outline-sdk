import buildWrapperAppTemplate from "./build_wrapper_app_template.mjs";
import minimist from "minimist";

(async function(){
  const args = minimist(process.argv.slice(2));

  if (!args.platform) {
    throw new Error(`Parameter \`--platform\` not provided.`);
  }

  if (!args.entryDomain) {
    throw new Error(`Parameter \`--entryDomain\` not provided.`);
  }

  await buildWrapperAppTemplate(args);
})();
