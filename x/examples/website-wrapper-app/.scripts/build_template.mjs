import handlebars from "handlebars";
import fs from "node:fs";
import path from "node:path";
import { glob } from "glob";

import startNavigationProxy from "./start_proxy.mjs";

const INPUT_DIR = "wrapper_template";
const OUTPUT_DIR = "output/wrapper_app";

(async function () {
  const { mainDomain, navigationUrl } = await startNavigationProxy();

  const sourceFilepaths = await glob(path.join(process.cwd(), INPUT_DIR, "**", "*"), {
    nodir: true,
    dot: true
  });

  for (const sourceFilepath of sourceFilepaths) {
    const destinationFilepath = path.join(
      process.cwd(),
      OUTPUT_DIR,
      path.relative(
        path.join(process.cwd(), INPUT_DIR),
        sourceFilepath
      )
    );

    // ensure directory
    fs.mkdirSync(path.dirname(destinationFilepath), { recursive: true });

    if (!sourceFilepath.endsWith(".handlebars")) {
      fs.copyFileSync(sourceFilepath, destinationFilepath);
      continue;
    }

    const template = handlebars.compile(fs.readFileSync(sourceFilepath, "utf8"));

    fs.writeFileSync(
      destinationFilepath.replace(/\.handlebars$/, ""),
      template({
        mainDomain,
        navigationUrl,
        whitelistedDomains: process.env.WHITELISTED_DOMAINS?.split(/,\s*/) ?? []
      }),
      "utf8"
    );

    console.log(`Generated ${destinationFilepath}`);
  }
})();
